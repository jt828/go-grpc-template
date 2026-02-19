package bootstrap

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"os"

	"github.com/jt828/go-grpc-template/pkg/snowflake"
	snowflakeImpl "github.com/jt828/go-grpc-template/pkg/snowflake/implementation"
)

func InitializeSnowflake() (snowflake.Snowflake, error) {
	nodeID, err := PodNodeID()
	if err != nil {
		return nil, err
	}
	return snowflakeImpl.NewSnowflake(nodeID)
}

func PodNodeID() (int64, error) {
	hostname := os.Getenv("HOSTNAME")
	if hostname == "" {
		return 0, fmt.Errorf("HOSTNAME is not set")
	}

	h := fnv.New64a()
	h.Write([]byte(hostname))
	nodeID := int64(binary.BigEndian.Uint64(h.Sum(nil)) % 1024)

	return nodeID, nil
}
