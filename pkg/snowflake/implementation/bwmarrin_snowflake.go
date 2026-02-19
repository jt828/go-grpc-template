package implementation

import (
	bwmarrin "github.com/bwmarrin/snowflake"
	"github.com/jt828/go-grpc-template/pkg/snowflake"
)

type bwmarrinSnowflake struct {
	node *bwmarrin.Node
}

func NewSnowflake(nodeID int64) (snowflake.Snowflake, error) {
	node, err := bwmarrin.NewNode(nodeID)
	if err != nil {
		return nil, err
	}
	return &bwmarrinSnowflake{node: node}, nil
}

func (s *bwmarrinSnowflake) Generate() int64 {
	return s.node.Generate().Int64()
}
