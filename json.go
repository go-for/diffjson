package diffjson

import (
	"github.com/go-for/ujson"
	"strconv"
)

var defaultDiffer = Differ{
	c: defaultConfig,
}

type Differ struct {
	c Config
}

func (d *Differ) Compare(old []byte, new []byte) (*Result, error) {
	decodeFunc := ujson.Unmarshal
	if d.c.GlobalIgnoreArrayOrder {
		decodeFunc = ujson.UnmarshalWithSort
	}
	j1, err := decodeFunc(old)
	if err != nil {
		return nil, err
	}
	j2, err := decodeFunc(new)
	if err != nil {
		return nil, err
	}
	result := d.diff(j1, j2, "")
	return result, nil
}

func Compare(old []byte, new []byte) (*Result, error) {
	return defaultDiffer.Compare(old, new)
}

type Result struct {
	Path            string
	Relation        Relation
	DataType        ujson.T
	Old             ujson.Any          // omit when DataType is Object
	New             ujson.Any          // omit when DataType is Object
	ObjectSubResult map[string]*Result `json:",omitempty"` // valid when DataType is TObject
	ArraySubResult  []*Result          `json:",omitempty"` // valid when DataType is TArray
}

type Relation string

const (
	ADD      Relation = "add"
	EQUAL    Relation = "equal"
	DELETE   Relation = "delete"
	REPLACE  Relation = "replace"
	MISMATCH Relation = "mismatch"
)

func (d *Differ) diff(m1, m2 ujson.Any, path string) *Result {
	// fmt.Printf("=====m1=====\n%#v\n\n=====m2=====\n%#v\n", m1, m2)
	if d.c.skip(path) {
		return nil
	}

	res := &Result{
		Path:     path,
		Relation: REPLACE,
		Old:      m1,
		New:      m2,
	}

	t1, t2 := m1.T(), m2.T()
	if t1 != t2 {
		res.Relation = MISMATCH
	}

	if d.c.GlobalIgnoreNumberType && t1.IsNumber() && t2.IsNumber() {
		if diffNumberLike(m1, m2) {
			res.Relation = EQUAL
			if d.c.omit(res) {
				return nil
			}
			return res
		}
	}

	switch o1 := m1.(type) {
	case ujson.Object:
		res.DataType = ujson.TObject
		res.Old = nil
		res.New = nil
		res.ObjectSubResult = make(map[string]*Result, 0)
		o2 := m2.(ujson.Object)
		// add
		for _, k := range o2.Keys() {
			p := objectPath(path, k)
			if d.c.skip(p) {
				continue
			}
			v2, _ := o2.Value(k)
			if v1, ok := o1.Value(k); !ok {
				res.ObjectSubResult[k] = &Result{
					Path:     p,
					Relation: ADD,
					Old:      v1,
					New:      v2,
				}
			}
		}
		// del
		for _, k := range o1.Keys() {
			p := objectPath(path, k)
			if d.c.skip(p) {
				continue
			}
			v1, _ := o1.Value(k)
			if v2, ok := o2.Value(k); !ok {
				res.ObjectSubResult[k] = &Result{
					Path:     p,
					Relation: DELETE,
					Old:      v1,
					New:      v2,
				}
			}
		}

		// other condition
		for _, k := range o1.Keys() {
			p := objectPath(path, k)
			if d.c.skip(p) {
				continue
			}
			v1, _ := o1.Value(k)
			if v2, ok := o2.Value(k); ok {
				subResult := d.diff(v1, v2, p)
				if d.c.omit(subResult) {
					continue
				}
				res.ObjectSubResult[k] = subResult
			}
		}

		deepEqual := true
		for _, r := range res.ObjectSubResult {
			if r.Relation != EQUAL {
				deepEqual = false
			}
		}
		if deepEqual {
			res.Relation = EQUAL
		}

	case ujson.Array:
		res.DataType = ujson.TArray
		res.Relation = REPLACE
		o2 := m2.(ujson.Array)
		if contains(d.c.IgnoreArrayOrderPath, path) {
			(&o1).Sort()
			(&o2).Sort()
		}
		if o1.Len() == o2.Len() {
			res.ArraySubResult = make([]*Result, o1.Len())
			deepEqual := true
			for i := 0; i < o1.Len(); i++ {
				p := arrayPath(path, i)
				if d.c.skip(p) {
					continue
				}
				subResult := d.diff(o1.Index(i), o2.Index(i), p)
				if d.c.omit(subResult) {
					continue
				}
				res.ArraySubResult[i] = subResult
				if subResult.Relation != EQUAL {
					deepEqual = false
				}
			}
			if deepEqual {
				res.Relation = EQUAL
			}
		}

	case ujson.NumberInt:
		res.DataType = ujson.TNumberInt
		res.Relation = REPLACE
		o2 := m2.(ujson.NumberInt)
		if o1.Int64() == o2.Int64() {
			res.Relation = EQUAL
		}
	case ujson.NumberUint:
		res.DataType = ujson.TNumberUint
		res.Relation = REPLACE
		o2 := m2.(ujson.NumberUint)
		if o1.Uint64() == o2.Uint64() {
			res.Relation = EQUAL
		}
	case ujson.NumberFloat:
		res.DataType = ujson.TNumberFloat
		res.Relation = REPLACE
		o2 := m2.(ujson.NumberFloat)
		if o1.Float64() == o2.Float64() {
			res.Relation = EQUAL
		}
	case ujson.String:
		res.DataType = ujson.TString
		res.Relation = REPLACE
		o2 := m2.(ujson.String)
		if o1.String() == o2.String() {
			res.Relation = EQUAL
		}
	case ujson.Bool:
		res.DataType = ujson.TBool
		res.Relation = REPLACE
		o2 := m2.(ujson.Bool)
		if o1.Bool() == o2.Bool() {
			res.Relation = EQUAL
		}
	case ujson.Null:
		res.Relation = EQUAL
	}

	if d.c.omit(res) {
		return nil
	}

	return res
}

func eqt(m1, m2 any) bool {
	switch t1 := m1.(type) {
	default:
		switch t2 := m2.(type) {
		default:
			if t1 == t2 {
				return true
			}
		}
	}
	return false
}

// double "comma ok" idiom
func commaOK(m1 ujson.Any, t1 ujson.Any, m2 ujson.Any, t2 ujson.Any) (o1 ujson.Any, ok1 bool, o2 ujson.Any, ok2 bool) {
	ms := [2]ujson.Any{m1, m2}
	ts := [2]ujson.Any{t1, t2}
	os := [2]ujson.Any{o1, o2}
	oks := [2]bool{ok1, ok2}
	for i := range ts {
		switch ts[i].(type) {
		case ujson.Object:
			os[i], oks[i] = ms[i].(ujson.Object)
		case ujson.Array:
			os[i], oks[i] = ms[i].(ujson.Array)
		case ujson.NumberInt:
			os[i], oks[i] = ms[i].(ujson.NumberInt)
		case ujson.NumberUint:
			os[i], oks[i] = ms[i].(ujson.NumberUint)
		case ujson.NumberFloat:
			os[i], oks[i] = ms[i].(ujson.NumberFloat)
		case ujson.String:
			os[i], oks[i] = ms[i].(ujson.String)
		case ujson.Bool:
			os[i], oks[i] = ms[i].(ujson.Bool)
		case ujson.Null:
			os[i], oks[i] = ms[i].(ujson.Null)
		}
	}
	return os[0], oks[0], os[1], oks[1]
}

func diffNumberLike(n1, n2 ujson.Number) bool {
	var vs = make([]float64, 2)
	for i, n := range []ujson.Number{n1, n2} {
		switch v := n.(type) {
		case ujson.NumberInt:
			vs[i] = float64(v.Int64())
		case ujson.NumberUint:
			vs[i] = float64(v.Uint64())
		case ujson.NumberFloat:
			vs[i] = v.Float64()
		}
	}
	return vs[0] == vs[1]
}

func objectPath(basePath, path string) string {
	if basePath == "" {
		return path
	}
	return basePath + "." + path
}

func arrayPath(basePath string, idx int) string {
	return basePath + "[#" + strconv.Itoa(idx+1) + "]"
}
