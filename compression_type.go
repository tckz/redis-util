package redisutil

//go:generate stringer -type=CompressionType compression_type.go
type CompressionType int

const (
	CompressionUnknown CompressionType = iota
	CompressionNone
	CompressionGzip
)

type CompressionInfo struct {
	Type CompressionType
	Ext  string
}

var compressionNone = CompressionInfo{Type: CompressionNone}

var compressionMap = map[string]CompressionInfo{
	"":     compressionNone,
	"none": compressionNone,
	"gzip": {Type: CompressionGzip, Ext: ".gz"},
	"gz":   {Type: CompressionGzip, Ext: ".gz"},
}

func GetCompressionType(compression string) CompressionInfo {
	if t, ok := compressionMap[compression]; !ok {
		return compressionNone
	} else {
		return t
	}
}
