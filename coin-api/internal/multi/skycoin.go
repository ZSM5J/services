package multi

import (
	"crypto/rand"
	"fmt"

	"bytes"

	"github.com/skycoin/services/coin-api/internal/locator"
	"github.com/skycoin/services/coin-api/internal/model"
	ehandler "github.com/skycoin/services/errhandler"
	"github.com/skycoin/skycoin/src/api/cli"
	"github.com/skycoin/skycoin/src/api/webrpc"
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/coin"
	"github.com/skycoin/skycoin/src/visor"
	"github.com/skycoin/skycoin/src/wallet"
)

// SkyСoinService provides generic access to various coins API
type SkyСoinService struct {
	// client interface{} // coin client API
	client *webrpc.Client
}

// NewSkyService returns new multicoin generic service
func NewSkyService(n *locator.Node) *SkyСoinService {
	s := &SkyСoinService{}
	s.client = &webrpc.Client{
		Addr: fmt.Sprintf("%s:%d", n.GetNodeHost(), n.GetNodePort()),
	}
	return s
}

func getRand() []byte {
	return cipher.RandByte(1024)
}
func getSeed() string {
	return cipher.SumSHA256(getRand()).Hex()
}

// GenerateAddr generates address, private keys, pubkeys from deterministic seed
func (s *SkyСoinService) GenerateAddr(count int, hideSecret bool) (*model.Response, error) {
	seed := getSeed()
	w, err := wallet.CreateAddresses(wallet.CoinTypeSkycoin, seed, count, false)
	if err != nil {
		return nil, err
	}

	entry := w.Entries[0]
	rsp := model.Response{
		Status: model.StatusOk,
		Code:   0,
		Result: &model.AddressResponse{
			Address: entry.Address,
		},
	}

	return &rsp, nil
}

// GenerateKeyPair generates key pairs
func (s *SkyСoinService) GenerateKeyPair() *model.Response {
	seed := getRand()
	rand.Read(seed)
	pub, sec := cipher.GenerateDeterministicKeyPair(seed)
	// address := cipher.AddressFromSecKey(sec)
	rsp := model.Response{
		Status: model.StatusOk,
		Code:   0,
		Result: &model.KeysResponse{
			Private: pub.Hex(),
			Public:  sec.Hex(),
		},
	}

	return &rsp
}

func getBalanceAddress(br *cli.BalanceResult) string {
	if len(br.Addresses) > 0 {
		return br.Addresses[0].Address
	}

	return ""
}

// CheckBalance check the balance (and get unspent outputs) for an address
func (s *SkyСoinService) CheckBalance(addr string) (*model.Response, error) {
	addressesToGetBalance := make([]string, 0, 1)
	addressesToGetBalance = append(addressesToGetBalance, addr)
	balanceResult, err := cli.GetBalanceOfAddresses(s.client, addressesToGetBalance)
	if err != nil {
		return nil, err
	}
	rsp := model.Response{
		Status: model.StatusOk,
		Code:   model.CodeNoError,
		Result: &model.BalanceResponse{
			Address: getBalanceAddress(balanceResult),
			Hours:   balanceResult.Spendable.Hours,
			Balance: balanceResult.Spendable.Coins,
			// balanceResult.
			// 	Coin: model.Coin{
			// //TODO: fill data here
			// },
		},
	}

	return &rsp, nil
}

// SignTransaction sign a transaction
func (s *SkyСoinService) SignTransaction(transid string) (rsp *model.Response, err error) {
	//TODO: VERIFY this sign transaction logic
	var buf bytes.Buffer
	buf.WriteString(transid)
	strbytes := buf.Bytes()
	var secKey cipher.SecKey
	defer func() {
		if r := recover(); r != nil {
			rsp.Status = model.StatusError
			rsp.Code = ehandler.RPCTransactionError
			rsp.Result = &model.TransactionSign{}
		}
	}()
	secKey = cipher.NewSecKey(strbytes)
	trans := coin.Transaction{
	//TODO: some creds here?
	}
	keysSec := make([]cipher.SecKey, 0, 1)
	keysSec = append(keysSec, secKey)
	trans.SignInputs(keysSec)
	//TODO: maybe we have to show all signatures?
	signid := trans.Sigs[0]
	rsp.Status = model.StatusOk
	rsp.Code = 0
	rsp.Result = &model.TransactionSign{
		Signid: signid.Hex(),
	}

	return rsp, nil
}

// CheckTransactionStatus check the status of a transaction (tracks transactions by transaction hash)
func (s *SkyСoinService) CheckTransactionStatus(txID string) (visor.TransactionStatus, error) {
	status, err := s.client.GetTransactionByID(txID)

	if err != nil {
		return visor.TransactionStatus{}, err
	}

	return status.Transaction.Status, nil
}

// InjectTransaction inject transaction into network
func (s *SkyСoinService) InjectTransaction(rawtx string) (*model.Response, error) {
	injectedT, err := s.client.InjectTransactionString(rawtx)
	if err != nil {
		return nil, err
	}
	statusT, err := s.client.GetTransactionByID(injectedT)
	if err != nil {
		return nil, err
	}

	var tStatus string
	if statusT.Transaction.Status.Confirmed {
		tStatus = "confirmed"
	} else {
		tStatus = "unconfirmed"
	}
	rsp := model.Response{
		Status: model.StatusOk,
		Code:   0,
		Result: &model.Transaction{
			Transid: injectedT,
			Status:  tStatus,
		},
	}

	return &rsp, nil
}