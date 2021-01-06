package contract

import (
	"math/big"
	"reflect"
	"strings"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreapi"
	"github.com/icon-project/goloop/service/scoreresult"
)

const (
	FUNC_PREFIX = "Ex_"
)

const (
	CID_CHAIN = "CID_CHAINSCORE"
)

type SystemScoreModule struct {
	New func(cid string, cc CallContext, from module.Address) (SystemScore, error)
}

var systemScoreModules = map[string]*SystemScoreModule{}

func RegisterSystemScore(id string, m *SystemScoreModule) {
	systemScoreModules[id] = m
}

type SystemScore interface {
	Install(param []byte) error
	Update(param []byte) error
	GetAPI() *scoreapi.Info
}

func getSystemScore(contentID string, cc CallContext, from module.Address) (score SystemScore, err error) {
	v, ok := systemScoreModules[contentID]
	if ok == false {
		return nil, scoreresult.ContractNotFoundError.Errorf(
			"ContractNotFound(cid=%s)", contentID)
	}
	return v.New(contentID, cc, from)
}

func CheckMethod(obj SystemScore) error {
	numMethod := reflect.ValueOf(obj).NumMethod()
	methodInfo := obj.GetAPI()
	invalid := false
	for i := 0; i < numMethod; i++ {
		m := reflect.TypeOf(obj).Method(i)
		if strings.HasPrefix(m.Name, FUNC_PREFIX) == false {
			continue
		}
		mName := strings.TrimPrefix(m.Name, FUNC_PREFIX)
		methodInfo := methodInfo.GetMethod(mName)
		if methodInfo == nil {
			continue
		}
		// CHECK INPUT
		numIn := m.Type.NumIn()
		if len(methodInfo.Inputs) != numIn-1 {
			return scoreresult.InvalidInstanceError.Errorf(
				"Wrong method input. method[%s]", mName)
		}
		var t reflect.Type
		for j := 1; j < numIn; j++ {
			t = m.Type.In(j)
			switch methodInfo.Inputs[j-1].Type {
			case scoreapi.Integer:
				if reflect.TypeOf(&common.HexInt{}) != t {
					invalid = true
				}
			case scoreapi.String:
				if reflect.TypeOf(string("")) != t {
					invalid = true
				}
			case scoreapi.Bytes:
				if reflect.TypeOf([]byte{}) != t {
					invalid = true
				}
			case scoreapi.Bool:
				if reflect.TypeOf(bool(false)) != t {
					invalid = true
				}
			case scoreapi.Address:
				if reflect.TypeOf(&common.Address{}).Implements(t) == false {
					invalid = true
				}
			default:
				invalid = true
			}
			if invalid == true {
				return scoreresult.InvalidInstanceError.Errorf(
					"wrong system score signature. method : %s, "+
						"expected input[%d] : %v BUT real type : %v", mName, j-1, methodInfo.Inputs[j-1].Type, t)
			}
		}

		numOut := m.Type.NumOut()
		if len(methodInfo.Outputs) != numOut-1 {
			return scoreresult.InvalidInstanceError.Errorf(
				"Wrong method output. method[%s]", mName)
		}
		for j := 0; j < len(methodInfo.Outputs); j++ {
			t := m.Type.Out(j)
			switch methodInfo.Outputs[j] {
			case scoreapi.Integer:
				if reflect.TypeOf(int(0)) != t && reflect.TypeOf(int64(0)) != t {
					invalid = true
				}
			case scoreapi.String:
				if reflect.TypeOf(string("")) != t {
					invalid = true
				}
			case scoreapi.Bytes:
				if reflect.TypeOf([]byte{}) != t {
					invalid = true
				}
			case scoreapi.Bool:
				if reflect.TypeOf(bool(false)) != t {
					invalid = true
				}
			case scoreapi.Address:
				if reflect.TypeOf(&common.Address{}).Implements(t) == false {
					invalid = true
				}
			case scoreapi.List:
				if t.Kind() != reflect.Slice && t.Kind() != reflect.Array {
					invalid = true
				}
			case scoreapi.Dict:
				if t.Kind() != reflect.Map {
					invalid = true
				}
			default:
				invalid = true
			}
			if invalid == true {
				return scoreresult.InvalidInstanceError.Errorf(
					"Wrong system score signature. method : %s, "+
						"expected output[%d] : %v BUT real type : %v", mName, j, methodInfo.Outputs[j], t)
			}
		}
	}
	return nil
}

func Invoke(score SystemScore, method string, paramObj *codec.TypedObj) (status error, result *codec.TypedObj, steps *big.Int) {
	defer func() {
		if err := recover(); err != nil {
			log.Debugf("Fail to sysCall method[%s]. err=%+v\n", method, err)
			status = scoreresult.UnknownFailureError.Errorf("Recover obj=%+v", err)
			result = nil
		}
	}()
	steps = big.NewInt(0)
	m := reflect.ValueOf(score).MethodByName(FUNC_PREFIX + method)
	if m.IsValid() == false {
		return scoreresult.ErrMethodNotFound, nil, steps
	}
	mType := m.Type()

	var params []interface{}
	if ps, err := common.DecodeAny(paramObj); err != nil {
		return scoreresult.ErrInvalidParameter, nil, steps
	} else {
		var ok bool
		params, ok = ps.([]interface{})
		if !ok {
			return scoreresult.ErrInvalidParameter, nil, steps
		}
	}

	if len(params) != mType.NumIn() {
		return scoreresult.ErrInvalidParameter, nil, steps
	}

	objects := make([]reflect.Value, len(params))
	for i, p := range params {
		oType := mType.In(i)
		pValue := reflect.ValueOf(p)
		if !pValue.IsValid() {
			objects[i] = reflect.New(mType.In(i)).Elem()
			continue
		}
		if !pValue.Type().AssignableTo(oType) {
			return scoreresult.ErrInvalidParameter, nil, steps
		}
		objects[i] = reflect.New(mType.In(i)).Elem()
		objects[i].Set(pValue)
	}

	r := m.Call(objects)
	rLen := len(r)

	if rLen > 0 {
		last := r[rLen-1].Interface()
		if last != nil {
			if err, ok := last.(error); ok {
				status = err
			} else {
				status = scoreresult.ErrInvalidInstance
			}
		}
		if rLen == 2 {
			result, status = common.EncodeAny(r[0].Interface())
		}
	}
	return
}
