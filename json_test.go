package diffjson

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"
)

func TestCompare(t *testing.T) {
	s1 := `{"test":123, "a":"s1", "b":100, "detail":{"arr":[1, "1", true]}}`
	s2 := `{"test":123, "a":"s2", "b":100.00, "detail":{"arr":[1, "1", false]}}`

	differ := Differ{c: Config{
		IgnorePath: []string{
			"a",
			"detail.arr[#3]",
		},
		GlobalIgnoreNumberType: true,
		OmitEqual:              true,
	}}
	res, err := differ.Compare([]byte(s1), []byte(s2))
	fmt.Printf("%#v\n", res)
	fmt.Printf("%v\n", err)

	s, _ := json.Marshal(res)
	fmt.Printf("%v\n", string(s))
}

func TestBigJson(t *testing.T) {
	f1, _ := os.Open("large-file1.json")
	f2, _ := os.Open("large-file2.json")
	s1, _ := io.ReadAll(f1)
	s2, _ := io.ReadAll(f2)
	differ := Differ{c: Config{
		IgnorePath: []string{
			"a",
			"detail.arr[#3]",
		},
		IgnoreArrayOrderPath: []string{
			"",
		},
		GlobalIgnoreArrayOrder: false,
		GlobalIgnoreNumberType: true,
		OmitEqual:              true,
	}}
	res, err := differ.Compare([]byte(s1), []byte(s2))
	fmt.Printf("%#v\n", res)
	fmt.Printf("%v\n", err)
	s, _ := json.Marshal(res)
	fmt.Printf("%v\n", string(s))
}
