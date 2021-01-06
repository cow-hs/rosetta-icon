package state

import (
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
)

const (
	VarStepPrice  = "step_price"
	VarStepCosts  = "step_costs"
	VarStepTypes  = "step_types"
	VarTreasury   = "treasury"
	VarGovernance = "governance"
	VarNetwork    = "network"
	VarChainID    = "chain_id"

	VarStepLimitTypes = "step_limit_types"
	VarStepLimit      = "step_limit"
	VarServiceConfig  = "serviceConfig"
	VarRevision       = "revision"
	VarMembers        = "members"
	VarDeployers      = "deployers"
	VarLicenses       = "licenses"
	VarTotalSupply    = "total_supply"

	VarTimestampThreshold = "timestamp_threshold"
	VarBlockInterval      = "block_interval"
	VarCommitTimeout      = "commit_timeout"
	VarRoundLimitFactor   = "round_limit_factor"
	VarMinimizeBlockGen   = "minimize_block_gen"
	VarTxHashToAddress    = "tx_to_address"
	VarDepositTerm        = "deposit_term"
)

const (
	DefaultNID = 1
)

const (
	SysConfigFee = 1 << iota
	SysConfigAudit
	SysConfigDeployerWhiteList
	SysConfigScorePackageValidator
	SysConfigMembership
	SysConfigMax
)

const (
	InfoBlockTimestamp = "B.timestamp"
	InfoBlockHeight    = "B.height"
	InfoTxHash         = "T.hash"
	InfoTxIndex        = "T.index"
	InfoTxTimestamp    = "T.timestamp"
	InfoTxNonce        = "T.nonce"
	InfoTxFrom         = "T.from"
	InfoRevision       = "Revision"
	InfoStepCosts      = "StepCosts"
	InfoContractOwner  = "C.owner"
)

const (
	SystemIDStr = "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"
)

var (
	SystemID      = []byte(SystemIDStr)
	SystemAddress = common.NewContractAddress(SystemID)
)

type WorldContext interface {
	WorldState
	Revision() module.Revision
	ToRevision(v int) module.Revision
	StepsFor(t StepType, n int) int64
	StepPrice() *big.Int
	BlockTimeStamp() int64
	GetStepLimit(t string) *big.Int
	BlockHeight() int64
	Treasury() module.Address
	Governance() module.Address
	GetInfo() map[string]interface{}
	WorldStateChanged(ws WorldState) WorldContext
	WorldVirtualState() WorldVirtualState
	GetFuture(lq []LockRequest) WorldContext
	SetTransactionInfo(ti *TransactionInfo)
	GetTransactionInfo(ti *TransactionInfo) bool
	TransactionID() []byte
	SetContractInfo(si *ContractInfo)
	DepositTerm() int64
	UpdateSystemInfo()

	IsDeployer(addr string) bool
	FeeEnabled() bool
	AuditEnabled() bool
	DeployerWhiteListEnabled() bool
	PackageValidatorEnabled() bool
	MembershipEnabled() bool
	TransactionTimestampThreshold() int64

	EnableSkipTransaction()
	SkipTransactionEnabled() bool
}

type BlockInfo struct {
	Timestamp int64
	Height    int64
}

type TransactionInfo struct {
	Group     module.TransactionGroup
	Index     int32
	Hash      []byte
	From      module.Address
	Timestamp int64
	Nonce     *big.Int
}

type ContractInfo struct {
	Owner module.Address
}

type worldContext struct {
	WorldState
	virtualState WorldVirtualState

	treasury   module.Address
	governance module.Address

	systemInfo systemStorageInfo

	blockInfo    BlockInfo
	txInfo       TransactionInfo
	contractInfo ContractInfo

	info map[string]interface{}

	skipTransaction bool

	platform Platform
}

func (c *worldContext) WorldVirtualState() WorldVirtualState {
	return c.virtualState
}

func (c *worldContext) GetFuture(lq []LockRequest) WorldContext {
	lq2 := make([]LockRequest, len(lq)+1)
	copy(lq2, lq)
	lq2[len(lq)] = LockRequest{
		Lock: AccountReadLock,
		ID:   SystemIDStr,
	}

	var wvs WorldVirtualState
	if c.virtualState != nil {
		wvs = c.virtualState.GetFuture(lq2)
	} else {
		wvs = NewWorldVirtualState(c.WorldState, lq2)
	}
	return c.WorldStateChanged(wvs)
}

// TODO What if some values such as deployer don't use cache here and are resolved on demand.
type systemStorageInfo struct {
	ass          AccountSnapshot
	stepPrice    *big.Int
	stepCosts    map[string]int64
	stepLimit    map[string]int64
	sysConfig    int64
	stepCostInfo *codec.TypedObj
	revision     module.Revision
	depositTerm  int64
}

func (si *systemStorageInfo) Update(wc *worldContext) bool {
	ass := wc.GetAccountSnapshot(SystemID)
	if si.ass != nil && !ass.StorageChangedAfter(si.ass) {
		return false
	}

	si.ass = ass

	as := scoredb.NewStateStoreWith(ass)
	revision := int(scoredb.NewVarDB(as, VarRevision).Int64())
	si.revision = wc.platform.ToRevision(revision)

	stepPrice := scoredb.NewVarDB(as, VarStepPrice).BigInt()
	si.stepPrice = stepPrice

	stepCosts := make(map[string]int64)
	stepTypes := scoredb.NewArrayDB(as, VarStepTypes)
	stepCostDB := scoredb.NewDictDB(as, VarStepCosts, 1)
	tcount := stepTypes.Size()
	for i := 0; i < tcount; i++ {
		tname := stepTypes.Get(i).String()
		if value := stepCostDB.Get(tname).Int64(); value != 0 {
			stepCosts[tname] = value
		}
	}
	si.stepCosts = stepCosts
	si.stepCostInfo = nil

	stepLimit := make(map[string]int64)
	stepLimitTypes := scoredb.NewArrayDB(as, VarStepLimitTypes)
	stepLimitDB := scoredb.NewDictDB(as, VarStepLimit, 1)
	tcount = stepLimitTypes.Size()
	for i := 0; i < tcount; i++ {
		tname := stepLimitTypes.Get(i).String()
		if value := stepLimitDB.Get(tname).Int64(); value != 0 {
			stepLimit[tname] = value
		}
	}
	si.stepLimit = stepLimit

	si.sysConfig = scoredb.NewVarDB(as, VarServiceConfig).Int64()
	si.depositTerm = scoredb.NewVarDB(as, VarDepositTerm).Int64()
	return true
}

func (c *worldContext) Revision() module.Revision {
	return c.systemInfo.revision
}

func (c *worldContext) DepositTerm() int64 {
	return c.systemInfo.depositTerm
}

func (c *worldContext) ToRevision(value int) module.Revision {
	return c.platform.ToRevision(value)
}

func (c *worldContext) StepsFor(t StepType, n int) int64 {
	if v, ok := c.systemInfo.stepCosts[string(t)]; ok {
		return v * int64(n)
	} else {
		return 0
	}
}

func (c *worldContext) StepPrice() *big.Int {
	return c.systemInfo.stepPrice
}

func (c *worldContext) GetStepLimit(t string) *big.Int {
	if v, ok := c.systemInfo.stepLimit[t]; ok {
		return big.NewInt(v)
	} else {
		return big.NewInt(0)
	}
}

func (c *worldContext) FeeEnabled() bool {
	if c.systemInfo.sysConfig&SysConfigFee == 0 {
		return false
	}
	return true
}

func (c *worldContext) AuditEnabled() bool {
	if c.systemInfo.sysConfig&SysConfigAudit == 0 {
		return false
	}
	return true
}

func (c *worldContext) DeployerWhiteListEnabled() bool {
	if c.systemInfo.sysConfig&SysConfigDeployerWhiteList == 0 {
		return false
	}
	return true
}

func (c *worldContext) PackageValidatorEnabled() bool {
	if c.systemInfo.sysConfig&SysConfigScorePackageValidator == 0 {
		return false
	}
	return true
}

func (c *worldContext) MembershipEnabled() bool {
	if c.systemInfo.sysConfig&SysConfigMembership == 0 {
		return false
	}
	return true
}

func (c *worldContext) TransactionTimestampThreshold() int64 {
	ass := c.GetAccountSnapshot(SystemID)
	as := scoredb.NewStateStoreWith(ass)
	tshInMS := scoredb.NewVarDB(as, VarTimestampThreshold).Int64()
	return tshInMS * 1000
}

func (c *worldContext) IsDeployer(addr string) bool {
	ass := c.GetAccountSnapshot(SystemID)
	as := scoredb.NewStateStoreWith(ass)
	db := scoredb.NewArrayDB(as, VarDeployers)
	if db.Size() > 0 {
		for i := 0; i < db.Size(); i++ {
			if addr == db.Get(i).Address().String() {
				return true
			}
		}
	}
	return false
}

func (c *worldContext) BlockTimeStamp() int64 {
	return c.blockInfo.Timestamp
}

func (c *worldContext) BlockHeight() int64 {
	return c.blockInfo.Height
}

func (c *worldContext) GetBlockInfo(bi *BlockInfo) {
	*bi = c.blockInfo
}

func (c *worldContext) Treasury() module.Address {
	return c.treasury
}

func (c *worldContext) Governance() module.Address {
	return c.governance
}

func tryVirtualState(ws WorldState) WorldVirtualState {
	wvs, _ := ws.(WorldVirtualState)
	return wvs
}

func (c *worldContext) WorldStateChanged(ws WorldState) WorldContext {
	wc := &worldContext{
		WorldState:   ws,
		virtualState: tryVirtualState(ws),
		treasury:     c.treasury,
		governance:   c.governance,
		systemInfo:   c.systemInfo,
		blockInfo:    c.blockInfo,
	}
	return wc
}

func (c *worldContext) SetTransactionInfo(ti *TransactionInfo) {
	c.txInfo = *ti
	c.info = nil
}

func (c *worldContext) GetTransactionInfo(ti *TransactionInfo) bool {
	if c.txInfo.Hash != nil {
		*ti = c.txInfo
		return true
	}
	return false
}

func (c *worldContext) TransactionID() []byte {
	return c.txInfo.Hash
}

func (c *worldContext) SetContractInfo(si *ContractInfo) {
	c.contractInfo = *si
	c.info = nil
}

func (c *worldContext) stepCostInfo() interface{} {
	if c.systemInfo.stepCostInfo == nil {
		c.systemInfo.stepCostInfo = common.MustEncodeAny(c.systemInfo.stepCosts)
	}
	return c.systemInfo.stepCostInfo
}

func (c *worldContext) GetInfo() map[string]interface{} {
	if c.info == nil {
		m := make(map[string]interface{})
		m[InfoBlockHeight] = c.blockInfo.Height
		m[InfoBlockTimestamp] = c.blockInfo.Timestamp
		m[InfoTxHash] = c.txInfo.Hash
		m[InfoTxIndex] = c.txInfo.Index
		m[InfoTxTimestamp] = c.txInfo.Timestamp
		m[InfoTxNonce] = c.txInfo.Nonce
		m[InfoTxFrom] = c.txInfo.From
		m[InfoRevision] = int(c.Revision())
		m[InfoStepCosts] = c.stepCostInfo()
		m[InfoContractOwner] = c.contractInfo.Owner
		c.info = m
	}
	return c.info
}

func (c *worldContext) EnableSkipTransaction() {
	c.skipTransaction = true
}

func (c *worldContext) SkipTransactionEnabled() bool {
	return c.skipTransaction
}

func (c *worldContext) UpdateSystemInfo() {
	if c.systemInfo.Update(c) {
		c.info = nil
	}
}

type Platform interface {
	ToRevision(value int) module.Revision
}

func NewWorldContext(ws WorldState, bi module.BlockInfo, plt Platform) WorldContext {
	var governance, treasury module.Address
	ass := ws.GetAccountSnapshot(SystemID)
	as := scoredb.NewStateStoreWith(ass)
	if as != nil {
		treasury = scoredb.NewVarDB(as, VarTreasury).Address()
		governance = scoredb.NewVarDB(as, VarGovernance).Address()
	}
	if treasury == nil {
		treasury = common.NewAddressFromString("hx1000000000000000000000000000000000000000")
	}
	if governance == nil {
		governance = common.NewAddressFromString("cx0000000000000000000000000000000000000001")
	}
	wc := &worldContext{
		WorldState:   ws,
		virtualState: tryVirtualState(ws),
		treasury:     treasury,
		governance:   governance,
		blockInfo:    BlockInfo{Timestamp: bi.Timestamp(), Height: bi.Height()},
		platform:     plt,
	}
	wc.UpdateSystemInfo()
	return wc
}
