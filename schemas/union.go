package schemas

type Union struct {
	Value interface{}
}

type UnionStruct struct {
	A *Union `ssz-max:"100"`
}

type HardUnionStruct struct {
	UnionStruct
}
