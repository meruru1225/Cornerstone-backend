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
	LibPath                  LibPathConfig           `mapstructure:"lib_path"`
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
	URL         string           `mapstructure:"url"`
	TextModel   string           `mapstructure:"text_model"`
	VisionModel string           `mapstructure:"vision_model"`
	ApiKey      string           `mapstructure:"api_key"`
	PromptsPath PromptPathConfig `mapstructure:"prompts_path"`
}

type PromptPathConfig struct {
	Chat            string `mapstructure:"chat"`
	ContentClassify string `mapstructure:"content_classify"`
	ContentSafe     string `mapstructure:"content_safe"`
	ImageSafe       string `mapstructure:"image_safe"`
	SearchChat      string `mapstructure:"search_chat"`
}

// MinIOConfig MinIO配置
type MinIOConfig struct {
	InternalEndpoint string `mapstructure:"internal_endpoint"`
	ExternalEndpoint string `mapstructure:"external_endpoint"`
	AccessKey        string `mapstructure:"access_key"`
	SecretKey        string `mapstructure:"secret_key"`
	MainBucket       string `mapstructure:"main_bucket"`
	TempBucket       string `mapstructure:"temp_bucket"`
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

// LibPathConfig 库路径
type LibPathConfig struct {
	FFmpeg       string `mapstructure:"ffmpeg"`
	FFprobe      string `mapstructure:"ffprobe"`
	Whisper      string `mapstructure:"whisper"`
	WhisperModel string `mapstructure:"whisper_model"`
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
