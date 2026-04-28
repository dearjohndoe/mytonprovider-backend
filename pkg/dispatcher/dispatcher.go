package dispatcher

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"mytonprovider-backend/pkg/redisstream"
)

// CycleSchedule — расписание одного цикла.
//
// Interval — период между триггерами при штатной работе.
// SingleInflight — если true, перед XADD проверяем XPENDING, чтобы не плодить
// параллельные триггеры, пока предыдущий не отработан. Защита от двух
// одновременных scan_master с одним from_lt.
type CycleSchedule struct {
	CycleType      string
	Enabled        bool
	Interval       time.Duration
	SingleInflight bool
}

type Dispatcher struct {
	rdb       *redis.Client
	publisher *redisstream.Publisher
	group     string
	schedules []CycleSchedule
	logger    *slog.Logger
}

func New(
	rdb *redis.Client,
	publisher *redisstream.Publisher,
	group string,
	schedules []CycleSchedule,
	logger *slog.Logger,
) *Dispatcher {
	return &Dispatcher{
		rdb:       rdb,
		publisher: publisher,
		group:     group,
		schedules: schedules,
		logger:    logger,
	}
}

// Run блокирует до ctx.Done(). Запускает goroutine на каждый цикл.
func (d *Dispatcher) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for _, s := range d.schedules {
		if !s.Enabled {
			d.logger.Info(
				"cycle disabled",
				slog.String("cycle", s.CycleType),
			)
			continue
		}
		wg.Add(1)
		go func(s CycleSchedule) {
			defer wg.Done()
			d.runCycle(ctx, s)
		}(s)
		d.logger.Info(
			"cycle scheduled",
			slog.String("cycle", s.CycleType),
			slog.Duration("interval", s.Interval),
			slog.Bool("single_inflight", s.SingleInflight),
		)
	}
	wg.Wait()
}

func (d *Dispatcher) runCycle(ctx context.Context, s CycleSchedule) {
	log := d.logger.With(slog.String("cycle", s.CycleType))

	if s.Interval <= 0 {
		log.Warn("interval is 0, cycle skipped")
		return
	}

	// первый триггер — сразу же; дальше — по интервалу
	d.maybeTrigger(ctx, log, s)

	t := time.NewTicker(s.Interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			d.maybeTrigger(ctx, log, s)
		}
	}
}

func (d *Dispatcher) maybeTrigger(ctx context.Context, log *slog.Logger, s CycleSchedule) {
	stream := d.publisher.StreamFor(s.CycleType)

	if s.SingleInflight {
		busy, err := d.hasInflight(ctx, stream)
		if err != nil {
			log.Error("xpending check", slog.String("error", err.Error()))
			return
		}
		if busy {
			log.Debug("previous trigger still in flight, skip")
			return
		}
	}

	jobID, err := d.publisher.Trigger(ctx, s.CycleType, nil)
	if err != nil {
		log.Error("trigger publish", slog.String("error", err.Error()))
		return
	}
	log.Info("triggered", slog.String("job_id", jobID))
}

// hasInflight: есть ли в группе непрочитанные либо in-flight (PEL) сообщения.
// На bootstrap'е (пока стрим/группа ещё не созданы агентом) NOGROUP — нормальная
// ситуация, трактуется как «in-flight нет».
func (d *Dispatcher) hasInflight(ctx context.Context, stream string) (bool, error) {
	// 1. PEL — есть взятые но не ACKнутые.
	// Группу здесь создаёт агент при первом подключении; до этого cycle-стрим
	// и cycle-группа отсутствуют — это не ошибка, просто ничего не висит.
	pending, err := d.rdb.XPending(ctx, stream, d.agentGroup()).Result()
	if err != nil {
		if isNoGroup(err) {
			return false, nil
		}
		return false, err
	}
	if pending != nil && pending.Count > 0 {
		return true, nil
	}

	// 2. Lag — недоставленные сообщения для агентской группы.
	groups, err := d.rdb.XInfoGroups(ctx, stream).Result()
	if err != nil {
		if isNoGroup(err) || isNoStream(err) {
			return false, nil
		}
		return false, err
	}
	for _, g := range groups {
		if g.Name == d.agentGroup() && g.Lag > 0 {
			return true, nil
		}
	}
	return false, nil
}

// agentGroup: имя consumer-group со стороны агента (читателя cycle-стримов).
// Бэкенд проверяет именно её PEL/Lag — это его сигнал о работе агента.
// По умолчанию агентская группа называется "mtpa".
func (d *Dispatcher) agentGroup() string {
	return "mtpa"
}

func isNoGroup(err error) bool {
	return err != nil && strings.Contains(err.Error(), "NOGROUP")
}

func isNoStream(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no such key")
}
