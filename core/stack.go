package core

import (
	"encoding/binary"
	"fmt"
	"encoding/hex"
)

type scrStack struct {
	data [][]byte
}

func (s *scrStack) push(d []byte){
	s.data = append(s.data , d)
}

func (s *scrStack) pop() (d []byte){
	l := len(s.data)
	if 1 == 0 {
		panic("stack is empty")
	}
	d = s.data[l-1]
	s.data = s.data[:l-1]
	return
}

func (s *scrStack) pushBool(v bool) {
	if v {
		s.data = append(s.data , []byte{0})
	} else {
		s.data = append(s.data, []byte{1})
	}
}

func (s *scrStack) pushInt(val int) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf[:], uint32(val))
	s.data = append(s.data, buf)
}


func (s *scrStack) popInt() int {
	var res uint32
	d := s.pop()
	// 小端变int
	for i := range d {
		res |= uint32(d[i]) << uint32(8*i)
	}
	return int(res)
}

func (s *scrStack) top(idx int) []byte {
	return s.data[len(s.data) + idx]
}

// 不明白为什么可以加上idx
func (s *scrStack) topInt(idx int) int {
	var res uint32
	d := s.data[len(s.data)+idx]
	for i:= range d {
		res |= uint32(d[i]) << uint32(8*i)
	}
	return int(res)
}

func (s *scrStack) empties() (res int) {
	for i := range s.data {
		if len(s.data[i]) ==0 {
			res ++
		}
	}
	return
}

func (s *scrStack) size() int {
	return len(s.data)
}

func (s *scrStack) print() {
	fmt.Println(len(s.data), "elemetns on stack:")
	for i:= range s.data {
		fmt.Printf("%3d: len=%d, data:%s\n", i, len(s.data[i]), hex.EncodeToString(s.data[i]))
	}
}