package providersmaster

import (
	"math/big"
	"strconv"
	"sync"
	"time"

	"github.com/xssnick/tonutils-go/tlb"

	tonclient "mytonprovider-backend/pkg/clients/ton"
	"mytonprovider-backend/pkg/constants"
	"mytonprovider-backend/pkg/models/db"
)

func fillStatuses(bagsStatuses *sync.Map, contracts []db.ContractToProviderRelation, reason constants.ReasonCode) {
	for _, sc := range contracts {
		bagsStatuses.Store(sc.ProviderAddress+sc.BagID, db.ContractProofsCheck{
			ContractAddress: sc.Address,
			ProviderAddress: sc.ProviderAddress,
			Reason:          reason,
		})
	}
}

func getKey(bagID, ip string, port int32) string {
	return ip + ":" + strconv.Itoa(int(port)) + "/" + bagID
}

func isRemovedByLowBalance(bagSize *big.Int, provider tonclient.Provider, contract tonclient.StorageContractProviders) bool {
	var storageFee = tlb.MustFromTON("0.05").Nano()

	mul := new(big.Int).Mul(new(big.Int).SetUint64(provider.RatePerMBDay), bagSize)
	mul = mul.Mul(mul, new(big.Int).SetUint64(uint64(provider.MaxSpan)))
	bounty := new(big.Int).Div(mul, big.NewInt(24*60*60*1024*1024))
	bounty = bounty.Add(bounty, storageFee)

	if new(big.Int).SetUint64(contract.Balance).Cmp(bounty) < 0 {
		var deadline int64
		fresh := provider.LastProofTime.Unix() <= 0
		if fresh {
			return false
		} else {
			deadline = provider.LastProofTime.Unix() + int64(provider.MaxSpan) + 3600
		}

		if time.Now().Unix() > deadline {
			return true
		}
	}

	return false
}
