package scoreapi

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreresult"
)

type MethodType int

const (
	Function MethodType = iota
	Fallback
	Event
)

func (t MethodType) String() string {
	switch t {
	case Function:
		return "function"
	case Fallback:
		return "fallback"
	case Event:
		return "eventlog"
	default:
		log.Panicf("Fail to convert MethodType=%d", t)
		return "Unknown"
	}
}

type TypeTag int

const (
	TUnknown TypeTag = iota
	TInteger
	TString
	TBytes
	TBool
	TAddress
	TList
	TDict
	TStruct
)

const (
	listDepthOffset = 4
	listDepthBits   = 4
	listDepthMask   = (1 << listDepthBits) - 1
	listDepthCheck  = listDepthMask << listDepthOffset
	maxListDepth    = listDepthMask

	valueTagBits = 4
	valueTagMask = (1 << valueTagBits) - 1
)

func (t TypeTag) String() string {
	switch t {
	case TInteger:
		return "int"
	case TString:
		return "str"
	case TBytes:
		return "bytes"
	case TBool:
		return "bool"
	case TAddress:
		return "Address"
	case TList:
		return "list"
	case TDict:
		return "dict"
	case TStruct:
		return "struct"
	default:
		return fmt.Sprintf("unknown(%d)", int(t))
	}
}

func (t TypeTag) ConvertJSONToTypedObj(bs []byte, fields []Field) (*codec.TypedObj, error) {
	var value interface{}
	switch t {
	case TInteger:
		var buffer common.HexInt
		if err := json.Unmarshal(bs, &buffer); err != nil {
			return nil, scoreresult.InvalidParameterError.Wrapf(err,
				"InvalidParameter(type=%s,json=%q)", t.String(), string(bs))
		}
		value = &buffer
	case TString:
		var buffer string
		if err := json.Unmarshal(bs, &buffer); err != nil {
			return nil, scoreresult.InvalidParameterError.Wrapf(err,
				"InvalidParameter(type=%s,json=%q)", t.String(), string(bs))
		}
		value = buffer
	case TBytes:
		var buffer common.HexBytes
		if err := json.Unmarshal(bs, &buffer); err != nil {
			return nil, scoreresult.InvalidParameterError.Wrapf(err,
				"InvalidParameter(type=%s,json=%q)", t.String(), string(bs))
		}
		value = buffer.Bytes()
	case TBool:
		var buffer common.HexInt32
		if err := json.Unmarshal(bs, &buffer); err != nil {
			return nil, scoreresult.InvalidParameterError.Wrapf(err,
				"InvalidParameter(type=%s,json=%q)", t.String(), string(bs))
		}
		if buffer.Value != 0 && buffer.Value != 1 {
			return nil, scoreresult.InvalidParameterError.Errorf(
				"InvalidParameter(type=%s,json=%q)", t.String(), string(bs))
		}
		value = buffer.Value != 0
	case TAddress:
		var buffer common.Address
		if err := json.Unmarshal(bs, &buffer); err != nil {
			return nil, scoreresult.InvalidParameterError.Wrapf(err,
				"InvalidParameter(type=%s,json=%q)", t.String(), string(bs))
		}
		value = &buffer
	case TStruct:
		buffer := make(map[string]*codec.TypedObj)
		var tmp map[string]json.RawMessage
		if err := json.Unmarshal(bs, &tmp); err != nil {
			return nil, scoreresult.InvalidParameterError.Wrapf(err,
				"InvalidParameter(type=%s,json=%q)", t.String(), string(bs))
		}
		for _, field := range fields {
			if bs, ok := tmp[field.Name]; ok {
				if obj, err := field.Type.ConvertJSONToTypedObj(bs, field.Fields, false); err != nil {
					return nil, err
				} else {
					buffer[field.Name] = obj
				}
			} else {
				return nil, scoreresult.InvalidParameterError.Errorf("InvalidParameterNoField(name=%s)", field.Name)
			}
		}
		value = buffer
	default:
		return nil, scoreresult.InvalidParameterError.Errorf("UnknownType(%s)", t.String())
	}
	if obj, err := common.EncodeAny(value); err != nil {
		return nil, scoreresult.InvalidParameterError.Wrapf(err,
			"InvalidParameter(type=%s,json=%q)", t.String(), string(bs))
	} else {
		return obj, nil
	}
}

func TypeTagOf(s string) TypeTag {
	switch s {
	case "bool":
		return TBool
	case "int":
		return TInteger
	case "str":
		return TString
	case "bytes":
		return TBytes
	case "Address":
		return TAddress
	case "list":
		return TList
	case "dict":
		return TDict
	case "struct":
		return TStruct
	default:
		return TUnknown
	}
}

type Field struct {
	Name   string
	Type   DataType
	Fields []Field
}

// DataType composed of following bits.
// ListDepth(4bits) + TypeTag(4bits)
type DataType int

const (
	Unknown DataType = iota
	Integer
	String
	Bytes
	Bool
	Address
	List
	Dict
	Struct
)

func (t DataType) Tag() TypeTag {
	return TypeTag(t & valueTagMask)
}

func (t DataType) ListDepth() int {
	return (int(t) >> listDepthOffset) & listDepthMask
}

func (t DataType) IsList() bool {
	return (t & listDepthCheck) != 0
}

func (t DataType) Elem() DataType {
	return t - (1 << listDepthOffset)
}

func ListTypeOf(depth int, t DataType) DataType {
	return t + (1<<listDepthOffset)*DataType(depth)
}

func (t DataType) String() string {
	prefix := strings.Repeat("[]", t.ListDepth())
	return prefix + t.Tag().String()
}

// DecodeJSO decode json object comes from JSON.
func (t DataType) ConvertJSONToTypedObj(bs []byte, fields []Field, nullable bool) (*codec.TypedObj, error) {
	if nullable && string(bs) == "null" {
		return codec.Nil, nil
	}

	if t.ListDepth() > 0 {
		var values []json.RawMessage
		if err := json.Unmarshal(bs, &values); err != nil {
			return nil, scoreresult.InvalidParameterError.Wrapf(err,
				"InvalidParameter(type=%s,json=%q)", t.String(), string(bs))
		}
		typed := make([]*codec.TypedObj, len(values))
		for i, v := range values {
			if tv, err := t.Elem().ConvertJSONToTypedObj(v, fields, false); err != nil {
				return nil, scoreresult.InvalidParameterError.Wrapf(err,
					"InvalidParameter(type=%s,json=%q)", t.String(), string(bs))
			} else {
				typed[i] = tv
			}
		}
		if obj, err := common.EncodeAny(typed); err != nil {
			return nil, scoreresult.InvalidParameterError.Wrapf(err,
				"InvalidParameter(type=%s,json=%q)", t.String(), string(bs))
		} else {
			return obj, nil
		}
	}

	return t.Tag().ConvertJSONToTypedObj(bs, fields)
}

// DecodeForJSON convert default bytes and event bytes into JSON value type.
func (t DataType) ConvertBytesToJSO(bs []byte) (interface{}, error) {
	if bs == nil {
		return nil, nil
	}
	if t.ListDepth() > 0 {
		return nil, errors.InvalidStateError.New("UnsupportedListType")
	}
	switch t.Tag() {
	case TInteger:
		var i common.HexInt
		i.SetBytes(bs)
		return &i, nil
	case TString:
		return string(bs), nil
	case TBytes:
		return common.HexBytes(bs), nil
	case TBool:
		if (len(bs) == 1 && bs[0] == 0) || len(bs) == 0 {
			return "0x0", nil
		} else {
			return "0x1", nil
		}
	case TAddress:
		addr := new(common.Address)
		if err := addr.SetBytes(bs); err != nil {
			return nil, err
		}
		return addr, nil
	default:
		return nil, errors.InvalidStateError.Errorf("UnsupportedType(type=%s)", t.String())
	}
}

// Decode convert default bytes into native type
func (t DataType) ConvertBytesToTypedObj(bs []byte) (*codec.TypedObj, error) {
	if bs == nil {
		return codec.Nil, nil
	}
	if t.ListDepth() > 0 {
		return nil, errors.IllegalArgumentError.Errorf("Unsupported Decoding type=%s", t.String())
	}
	switch t.Tag() {
	case TInteger:
		var i common.HexInt
		if len(bs) > 0 {
			i.SetBytes(bs)
		}
		return common.EncodeAny(&i)
	case TString:
		return common.EncodeAny(string(bs))
	case TBytes:
		return common.EncodeAny(bs)
	case TBool:
		if (len(bs) == 1 && bs[0] == 0) || len(bs) == 0 {
			return common.EncodeAny(false)
		} else {
			return common.EncodeAny(true)
		}
	case TAddress:
		addr := new(common.Address)
		if err := addr.SetBytes(bs); err != nil {
			return nil, err
		}
		return common.EncodeAny(addr)
	default:
		return nil, errors.IllegalArgumentError.Errorf("Unsupported Decoding type=%s", t.String())
	}
}

// ValidateBytes validate event bytes.
func (t DataType) ValidateEvent(bs []byte) error {
	if bs == nil {
		return nil
	}
	if t.ListDepth() > 0 {
		return errors.InvalidStateError.Errorf("InvalidType(type=%s)", t.String())
	}
	switch t.Tag() {
	case TInteger:
		if len(bs) == 0 {
			return errors.IllegalArgumentError.New("InvalidIntegerBytes")
		}
	case TBool:
		if len(bs) != 1 {
			return errors.IllegalArgumentError.Errorf("InvalidBoolBytes(bs=<%#x>)", bs)
		}
		if bs[0] > 1 {
			return errors.IllegalArgumentError.Errorf("InvalidBoolBytes(bs=<%#x>)", bs)
		}
	case TAddress:
		var addr common.Address
		if err := addr.SetBytes(bs); err != nil {
			return errors.IllegalArgumentError.New("InvalidAddressBytes")
		}
	case TString:
		if !utf8.Valid(bs) {
			return errors.IllegalArgumentError.New("InvalidUTF8Chars")
		}
	case TStruct:
		return errors.InvalidStateError.Errorf("InvalidType(type=%s)", t.String())
	}
	return nil
}

var inputTypeTag = map[TypeTag]uint8{
	TInteger: common.TypeInt,
	TString:  codec.TypeString,
	TBytes:   codec.TypeBytes,
	TBool:    codec.TypeBool,
	TAddress: common.TypeAddress,
}

var outputTypeTag = map[TypeTag]struct {
	tag      uint8
	nullable bool
}{
	TInteger: {common.TypeInt, false},
	TString:  {codec.TypeString, false},
	TBytes:   {codec.TypeBytes, true},
	TBool:    {codec.TypeBool, false},
	TAddress: {common.TypeAddress, true},
	TList:    {codec.TypeList, true},
	TDict:    {codec.TypeDict, true},
}

func (t DataType) ValidateInput(obj *codec.TypedObj, fields []Field, nullable bool) error {
	if obj == nil {
		obj = codec.Nil
	}
	if obj.Type == codec.TypeNil && nullable {
		return nil
	}
	if t.ListDepth() > 0 {
		if codec.TypeList != obj.Type {
			return errors.IllegalArgumentError.Errorf(
				"InvalidType(exp=list,type=%d)", obj.Type)
		}
		children, ok := obj.Object.([]*codec.TypedObj)
		if !ok {
			return errors.IllegalArgumentError.Errorf(
				"InvalidValue(exp=[]*codec.TypedObj,real=%T)", obj.Object)
		}
		for _, child := range children {
			if err := t.Elem().ValidateInput(child, fields, false); err != nil {
				return err
			}
		}
		return nil
	}
	if t.Tag() == TStruct {
		if obj.Type != codec.TypeDict {
			return errors.IllegalArgumentError.Errorf(
				"InvalidType(exp=TypeDict,real=%d)", obj.Type)
		}
		childMap, ok := obj.Object.(map[string]*codec.TypedObj)
		if !ok {
			return errors.IllegalArgumentError.Errorf(
				"InvalidValue(exp=[]*codec.TypedObj,real=%T)", obj.Object)
		}
		for _, field := range fields {
			if value, ok := childMap[field.Name]; ok {
				if err := field.Type.ValidateInput(value, field.Fields, false); err != nil {
					return err
				}
			} else {
				return errors.IllegalArgumentError.Errorf("NoValueForField(field=%s)", field.Name)
			}
		}
		if len(childMap) > len(fields) {
			return errors.IllegalArgumentError.Errorf(
				"UnexpectedFields(n=%d)", len(childMap)-len(fields))
		}
		return nil
	}
	if typeTag, ok := inputTypeTag[t.Tag()]; !ok {
		return errors.IllegalArgumentError.Errorf("InvalidType(%s)", t.Tag().String())
	} else {
		if typeTag == obj.Type {
			return nil
		}
		return errors.IllegalArgumentError.Errorf(
			"InvalidType(exp=%d,type=%d)", typeTag, obj.Type)
	}
	return nil
}

func (t DataType) ValidateOutput(obj *codec.TypedObj) error {
	if obj == nil {
		obj = codec.Nil
	}
	if t.ListDepth() > 0 {
		return errors.InvalidStateError.Errorf("InvalidTypeForOutput(%s)", t.String())
	}
	if typeTag, ok := outputTypeTag[t.Tag()]; !ok {
		return errors.IllegalArgumentError.Errorf("InvalidType(%s)", t.Tag().String())
	} else {
		if typeTag.tag == obj.Type {
			return nil
		}
		if obj.Type == codec.TypeNil && typeTag.nullable {
			return nil
		}
		return errors.IllegalArgumentError.Errorf(
			"InvalidType(exp=%d,type=%d)", typeTag.tag, obj.Type)
	}
}

// DataTypeOf returns type for the specified name.
func DataTypeOf(s string) DataType {
	depth := 0
	for strings.HasPrefix(s, "[]") {
		depth += 1
		s = s[2:]
	}
	if depth > maxListDepth {
		return Unknown
	}
	tag := TypeTagOf(s)
	if tag == TUnknown {
		return Unknown
	}
	return ListTypeOf(depth, DataType(tag))
}

const (
	FlagReadOnly = 1 << iota
	FlagExternal
	FlagPayable
	FlagIsolated
)

type Parameter struct {
	Name    string
	Type    DataType
	Default []byte
	Fields  []Field
}

func (p *Parameter) RLPEncodeSelf(e codec.Encoder) error {
	e2, err := e.EncodeList()
	if err != nil {
		return err
	}
	if err = e2.EncodeMulti(p.Name, p.Type, p.Default); err != nil {
		return err
	}
	if len(p.Fields) > 0 {
		return e2.Encode(p.Fields)
	}
	return nil
}

func (p *Parameter) RLPDecodeSelf(d codec.Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}
	if cnt, err := d2.DecodeMulti(&p.Name, &p.Type, &p.Default, &p.Fields); err == nil || err == io.EOF {
		if cnt < 3 {
			return errors.Wrap(codec.ErrInvalidFormat, "LessItems")
		}
		return nil
	} else {
		return err
	}
}

type Method struct {
	Type    MethodType
	Name    string
	Flags   int
	Indexed int
	Inputs  []Parameter
	Outputs []DataType
}

func (a *Method) IsPayable() bool {
	return a.Type != Event && (a.Flags&FlagPayable) != 0
}

func (a *Method) IsReadOnly() bool {
	return a.Type == Function && (a.Flags&FlagReadOnly) != 0
}

func (a *Method) IsExternal() bool {
	return a.Type == Function && (a.Flags&(FlagExternal|FlagReadOnly)) != 0
}

func (a *Method) IsIsolated() bool {
	return a.Type != Event && (a.Flags&FlagIsolated) != 0
}

func (a *Method) IsCallable() bool {
	return a.Type != Event
}

func (a *Method) IsFallback() bool {
	return a.Type == Fallback
}

func (a *Method) IsEvent() bool {
	return a.Type == Event
}
func (a *Method) ToJSON(version module.JSONVersion) (interface{}, error) {
	m := make(map[string]interface{})
	m["type"] = a.Type.String()
	m["name"] = a.Name

	inputs := make([]interface{}, len(a.Inputs))
	for i, input := range a.Inputs {
		io := make(map[string]interface{})
		io["name"] = input.Name
		io["type"] = input.Type.String()
		if a.Type == Event {
			if i < a.Indexed {
				io["indexed"] = "0x1"
			}
		} else {
			if i >= a.Indexed {
				if def, err := input.Type.ConvertBytesToJSO(input.Default); err == nil {
					io["default"] = def
				} else {
					log.Warnf("Fail to decode default bytes err=%+v", def)
				}
			}
		}
		inputs[i] = io
	}
	m["inputs"] = inputs

	outputs := make([]interface{}, len(a.Outputs))
	for i, output := range a.Outputs {
		oo := make(map[string]interface{})
		oo["type"] = output.String()
		outputs[i] = oo
	}
	m["outputs"] = outputs
	if (a.Flags & FlagReadOnly) != 0 {
		m["readonly"] = "0x1"
	}
	if (a.Flags & FlagPayable) != 0 {
		m["payable"] = "0x1"
	}
	if (a.Flags & FlagIsolated) != 0 {
		m["isolated"] = "0x1"
	}
	return m, nil
}

func (a *Method) EnsureParamsSequential(paramObj *codec.TypedObj) (*codec.TypedObj, error) {
	if paramObj.Type == codec.TypeList {
		tol := paramObj.Object.([]*codec.TypedObj)
		if len(tol) < a.Indexed {
			return nil, scoreresult.InvalidParameterError.Errorf(
				"NotEnoughParameters(given=%d,required=%d)", len(tol), a.Indexed)
		}
		if len(tol) > len(a.Inputs) {
			return nil, scoreresult.InvalidParameterError.Errorf(
				"TooManyParameters(given=%d,all=%d)", len(tol), len(a.Inputs))
		}
		tolNew := tol
		for i, input := range a.Inputs {
			inputType := a.Inputs[i].Type
			if i < len(tol) {
				to := tol[i]
				nullable := (i >= a.Indexed) && input.Default == nil
				if err := inputType.ValidateInput(to, input.Fields, nullable); err != nil {
					return nil, err
				}
			} else {
				if obj, err := inputType.ConvertBytesToTypedObj(input.Default); err != nil {
					return nil, err
				} else {
					tolNew = append(tolNew, obj)
				}
			}
		}
		paramObj.Object = tolNew
		return paramObj, nil
	}

	if paramObj.Type != codec.TypeDict {
		return nil, scoreresult.ErrInvalidParameter
	}
	params, ok := paramObj.Object.(map[string]*codec.TypedObj)
	if !ok {
		return nil, scoreresult.InvalidParameterError.Errorf(
			"FailToCastDictToMap(type=%[1]T, obj=%+[1]v)", paramObj.Object)
	}
	inputs := make([]*codec.TypedObj, len(a.Inputs))
	for i, input := range a.Inputs {
		if obj, ok := params[input.Name]; ok {
			nullable := (i >= a.Indexed) && input.Default == nil
			if err := input.Type.ValidateInput(obj, input.Fields, nullable); err != nil {
				return nil, scoreresult.InvalidParameterError.Wrapf(err,
					"InvalidParameter(exp=%s, value=%T)", input.Type, obj)
			}
			inputs[i] = obj
		} else {
			if i >= a.Indexed {
				if obj, err := input.Type.ConvertBytesToTypedObj(input.Default); err != nil {
					return nil, scoreresult.InvalidParameterError.Wrapf(err,
						"InvalidParameter(exp=%s, value=%T)", input.Type, obj)
				} else {
					inputs[i] = obj
				}
			} else {
				return nil, scoreresult.InvalidParameterError.Errorf(
					"MissingParameter(name=%s)", input.Name)
			}
		}
	}
	return common.MustEncodeAny(inputs), nil
}

func (a *Method) Signature() string {
	args := make([]string, len(a.Inputs))
	for i := 0; i < len(args); i++ {
		args[i] = a.Inputs[i].Type.String()
	}
	return fmt.Sprintf("%s(%s)", a.Name, strings.Join(args, ","))
}

func (a *Method) CheckEventData(indexed [][]byte, data [][]byte) error {
	if len(indexed)+len(data) != len(a.Inputs)+1 {
		return IllegalEventError.Errorf(
			"InvalidEventData(exp=%d,given=%d)",
			len(a.Inputs)+1, len(indexed)+len(data))
	}
	if len(indexed) != a.Indexed+1 {
		return IllegalEventError.Errorf(
			"InvalidIndexCount(exp=%d,given=%d)", a.Indexed, len(indexed)-1)
	}
	for i, p := range a.Inputs {
		var input []byte
		if i < len(indexed)-1 {
			input = indexed[i+1]
		} else {
			input = data[i+1-len(indexed)]
		}
		if err := p.Type.ValidateEvent(input); err != nil {
			return IllegalEventError.Wrapf(err,
				"IllegalEvent(sig=%s,idx=%d,data=0x%#x)",
				a.Signature(), i, input)
		}
	}
	return nil
}

func (a *Method) ConvertParamsToTypedObj(bs []byte) (*codec.TypedObj, error) {
	var params map[string]json.RawMessage
	if len(bs) > 0 {
		if err := json.Unmarshal(bs, &params); err != nil {
			return nil, scoreresult.WithStatus(err, module.StatusInvalidParameter)
		}
	}
	matched := 0
	inputs := make([]*codec.TypedObj, len(a.Inputs))
	for i, input := range a.Inputs {
		param, ok := params[input.Name]
		if !ok {
			if i >= a.Indexed {
				if obj, err := input.Type.ConvertBytesToTypedObj(input.Default); err != nil {
					return nil, scoreresult.InvalidParameterError.Wrapf(err,
						"InvalidParameter(exp=%s, value=%T)", input.Type, obj)
				} else {
					inputs[i] = obj
				}
				continue
			}
			return nil, scoreresult.Errorf(module.StatusInvalidParameter,
				"MissingParam(param=%s)", input.Name)
		}
		matched += 1
		if obj, err := input.Type.ConvertJSONToTypedObj(param, input.Fields, false); err != nil {
			return nil, err
		} else {
			inputs[i] = obj
		}
	}

	if matched != len(params) {
		return nil, scoreresult.Errorf(module.StatusInvalidParameter,
			"UnexpectedParam(%v)\n", params)
	}

	if to, err := common.EncodeAny(inputs); err != nil {
		return nil, scoreresult.WithStatus(err, module.StatusInvalidParameter)
	} else {
		return to, nil
	}
}

func (a *Method) EnsureResult(result *codec.TypedObj) error {
	if a == nil {
		return scoreresult.MethodNotFoundError.New("NoMethod")
	}
	if result == nil {
		result = codec.Nil
	}
	if len(a.Outputs) == 0 {
		if result.Type == codec.TypeNil {
			return nil
		}
		if !a.IsReadOnly() {
			// Some of execution environment returns empty
			// outputs for writable functions with outputs.
			// To support old versions, it ignores
			// empty outputs.
			return nil
		}
		return scoreresult.UnknownFailureError.Errorf(
			"InvalidReturn(exp=None,real=%d)", result.Type)
	}
	var results []*codec.TypedObj
	if len(a.Outputs) == 1 {
		results = []*codec.TypedObj{result}
	} else {
		if result.Type != codec.TypeList {
			return scoreresult.UnknownFailureError.Errorf(
				"InvalidReturnType(type=%d)", result.Type)
		}
		if rs, ok := result.Object.([]*codec.TypedObj); !ok {
			return scoreresult.UnknownFailureError.Errorf(
				"InvalidReturnType(type=%T)", result.Object)
		} else {
			results = rs
		}
	}
	if len(a.Outputs) != len(results) {
		return scoreresult.UnknownFailureError.Errorf(
			"InvalidReturnLength(exp=%d,real=%d)",
			len(a.Outputs), len(results))
	}
	for i, o := range results {
		if err := a.Outputs[i].ValidateOutput(o); err != nil {
			return scoreresult.UnknownFailureError.Wrapf(err,
				"InvalidReturnType(idx=%d)", i)
		}
	}
	return nil
}
