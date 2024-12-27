package godbf

// DBFHeader represents the structure of the DBFHandler file header.
type DBFHeader struct {
	Version          byte
	LastUpdateYear   byte
	LastUpdateMonth  byte
	LastUpdateDay    byte
	NumRecords       uint32
	HeaderLength     uint16
	RecordLength     uint16
	Reserved         [2]byte
	Flag             byte
	EncryptFlag      byte
	Reserved2        [12]byte
	MDXFlag          byte
	LanguageDriverID byte
	Reserved3        [2]byte
}

// FieldDescriptor represents the structure of a field descriptor in a DBFHandler file.
type FieldDescriptor struct {
	Name       [11]byte
	Type       byte
	Reserved1  [4]byte
	Length     byte
	Decimal    byte
	Reserved2  [2]byte
	WorkAreaID byte
	Reserved3  [10]byte
	Flag       byte
}
