package scoredb

import (
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/service/scoreresult"
)

type ArrayDB struct {
	key   []byte
	size  WritableValue
	store StateStore
}

func NewArrayDB(store StateStore, keys ...interface{}) *ArrayDB {
	kbytes := ToKey(ArrayDBPrefix, keys...)

	return &ArrayDB{
		key:   kbytes,
		size:  NewValueFromStore(store, crypto.SHA3Sum256(kbytes)),
		store: store,
	}
}

func (a *ArrayDB) Size() int {
	return int(a.size.Int64())
}

func (a *ArrayDB) keyHashForIndex(i int) []byte {
	return crypto.SHA3Sum256(AppendKeys(a.key, i))
}

func (a *ArrayDB) Get(i int) Value {
	if i < 0 || i >= a.Size() {
		return nil
	}
	bs, err := a.store.GetValue(a.keyHashForIndex(i))
	if err != nil || bs == nil {
		return nil
	}
	return NewValueFromBytes(bs)
}

func (a *ArrayDB) Set(i int, v interface{}) error {
	if i < 0 || i >= a.Size() {
		return scoreresult.ErrInvalidContainerAccess
	}
	return must(a.store.SetValue(a.keyHashForIndex(i), ToBytes(v)))
}

func (a *ArrayDB) Put(v interface{}) error {
	idx := a.Size()
	if err := must(a.store.SetValue(a.keyHashForIndex(idx), ToBytes(v))); err != nil {
		return err
	}
	return a.size.Set(idx + 1)
}

func (a *ArrayDB) Pop() Value {
	idx := a.Size()
	if idx == 0 {
		return nil
	}
	khash := a.keyHashForIndex(idx - 1)
	var bs []byte
	if ov, err := a.store.DeleteValue(khash); err != nil {
		log.Panicf("Fail to delete last element")
	} else {
		bs = ov
	}
	if idx > 1 {
		if err := a.size.Set(idx - 1); err != nil {
			log.Panicf("Fail to update size")
		}
	} else {
		if err := a.size.Delete(); err != nil {
			log.Panicf("Fail to delete size")
		}
	}
	return NewValueFromBytes(bs)
}
