package snowflake

type Snowflake interface {
	Generate() int64
}
