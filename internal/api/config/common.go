package config

// Config 配置主体
type Config struct {
	Server                   ServerConfig            `mapstructure:"server"`
	DB                       DBConfig                `mapstructure:"database"`
	Redis                    RedisConfig             `mapstructure:"redis"`
	SMS                      SMSConfig               `mapstructure:"sms"`
	LLM                      LLMConfig               `mapstructure:"llm"`
	MinIO                    MinIOConfig             `mapstructure:"minio"`
	Elastic                  ElasticConfig           `mapstructure:"elastic"`
	Kafka                    KafkaConfig             `mapstructure:"kafka"`
	KafkaUserConsumer        KafkaUserConsumer       `mapstructure:"kafka_user_consumer"`
	KafkaUserDetailConsumer  KafkaUserDetailConsumer `mapstructure:"kafka_user_detail_consumer"`
	KafkaUserFollowsConsumer KafkaUserFollowConsumer `mapstructure:"kafka_user_follow_consumer"`
	KafkaPostConsumer        KafkaPostConsumer       `mapstructure:"kafka_post_consumer"`
}

// ServerConfig Server配置
type ServerConfig struct {
	Port int `mapstructure:"port"`
}

// DBConfig 数据库配置
type DBConfig struct {
	DSN         string `mapstructure:"dsn"`
	MaxIdle     int    `mapstructure:"max_idle"`
	MaxOpen     int    `mapstructure:"max_open"`
	MaxLifetime int    `mapstructure:"max_lifetime"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

type SMSConfig struct {
	URL      string `mapstructure:"url"`
	Username string `mapstructure:"username"`
	ApiKey   string `mapstructure:"api_key"`
}

type LLMConfig struct {
	URL    string `mapstructure:"url"`
	Model  string `mapstructure:"model"`
	ApiKey string `mapstructure:"api_key"`
}

// MinIOConfig MinIO配置
type MinIOConfig struct {
	InternalEndpoint string `mapstructure:"internal_endpoint"`
	ExternalEndpoint string `mapstructure:"external_endpoint"`
	AccessKey        string `mapstructure:"access_key"`
	SecretKey        string `mapstructure:"secret_key"`
	Bucket           string `mapstructure:"bucket"`
	InternalUseSSL   bool   `mapstructure:"internal_use_ssl"`
	UsePublicLink    bool   `mapstructure:"use_public_link"`
}

// ElasticConfig Elastic配置
type ElasticConfig struct {
	Address  string         `mapstructure:"address"`
	Username string         `mapstructure:"username"`
	Password string         `mapstructure:"password"`
	Indices  ElasticIndices `mapstructure:"indices"`
}

// ElasticIndices Elastic索引
type ElasticIndices struct {
	UserIndex string `mapstructure:"user_index"`
	PostIndex string `mapstructure:"post_index"`
}

type KafkaConfig struct {
	Brokers  []string       `mapstructure:"brokers"`
	Sasl     SaslConfig     `mapstructure:"sasl"`
	Consumer ConsumerConfig `mapstructure:"consumer"`
}

type SaslConfig struct {
	Enable   bool   `mapstructure:"enable"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type ConsumerConfig struct {
	SessionTimeout    int `mapstructure:"session_timeout"`
	HeartbeatInterval int `mapstructure:"heartbeat_interval"`
	RebalanceTimeout  int `mapstructure:"rebalance_timeout"`
}

type KafkaUserConsumer struct {
	Topic   string `mapstructure:"topic"`
	GroupID string `mapstructure:"group_id"`
}

type KafkaUserDetailConsumer struct {
	Topic   string `mapstructure:"topic"`
	GroupID string `mapstructure:"group_id"`
}

type KafkaUserFollowConsumer struct {
	Topic   string `mapstructure:"topic"`
	GroupID string `mapstructure:"group_id"`
}

type KafkaPostConsumer struct {
	Topic   string `mapstructure:"topic"`
	GroupID string `mapstructure:"group_id"`
}
