package conf

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/spf13/viper"
)

var runBot = flag.Bool("run_bot", false, "run bot")
var runConf = flag.String("run_conf", "", "run config")

// Get_RUN_CONF 获取 RUN_CONF 环境变量，优先从环境变量读取，如果没有则从命令行参数读取
func Get_RUN_CONF() string {
	// 先从环境变量读取
	if envRunConf := os.Getenv("RUN_CONF"); envRunConf != "" {
		return envRunConf
	}
	// 如果环境变量没有，则从命令行参数读取
	return *runConf
}

type Config struct {
	ServerConf       *Server       `mapstructure:"server"`
	MysqlMasterConf  *Mysql        `mapstructure:"db"`
	RedisConf        *Redis        `mapstructure:"redis"`
	NotificationConf *Notification `mapstructure:"notification"`
}

// Server config
type Server struct {
	ServiceName string `mapstructure:"service_name"`
	RunMode     string `mapstructure:"run_mode"`
	Port        string `mapstructure:"port"`
	Env         string `mapstructure:"env"`
	ProductName string `mapstructure:"product_name"`
	ApiHost     string `mapstructure:"api_host"`
	MachineName string `mapstructure:"machine_name"`
}

// Mysql config
type Mysql struct {
	User        string `mapstructure:"user"`
	Password    string `mapstructure:"password"`
	Host        string `mapstructure:"host"`
	DbName      string `mapstructure:"db_name"`
	MaxIdleConn int    `mapstructure:"max_idle_conn"`
	MaxOpenConn int    `mapstructure:"max_open_conn"`
	LogMode     bool   `mapstructure:"log_mode"`
}

// Redis config
type Redis struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	UserName string `mapstructure:"username"`
	DB       int    `mapstructure:"db"`
	TLS      bool   `mapstructure:"tls"`
	PoolSize int    `mapstructure:"pool_size"`
	MinIdle  int    `mapstructure:"min_idle"`
}

// Notification 通知配置
type Notification struct {
	Lark LarkNotification `mapstructure:"lark"`
}

// LarkNotification Lark 通知配置（全节点共用一套 webhook）
type LarkNotification struct {
	OrderWebhookURL  string `mapstructure:"order_webhook_url"`  // 下单结果专用机器人 Webhook URL
	AlertWebhookURL  string `mapstructure:"alert_webhook_url"`  // 异常报警机器人 Webhook URL
	NotifyWebhookURL string `mapstructure:"notify_webhook_url"` // 通知消息机器人 Webhook URL
	ServiceName      string `mapstructure:"service_name"`       // 服务名称
	Timeout          int    `mapstructure:"timeout"`            // 请求超时时间(秒)
}

var GlobalConf *Config

func NewCfg() *Config {
	// 获取 run_conf 值（优先从环境变量读取）
	runConfValue := Get_RUN_CONF()
	if runConfValue == "" {
		log.Fatal("run_conf parameter is required (set RUN_CONF environment variable or use --run_conf flag)")
	}

	switch runConfValue {
	case "local":
		return loadLocalConfig("conf/local.conf.yaml")
	case "dev":
		return loadLocalConfig("conf/dev.conf.yaml")
	case "test":
		return loadLocalConfig("conf/test.conf.yaml")
	case "prod":
		return loadLocalConfig("conf/prod.conf.yaml")
	default:
		return loadLocalConfig("conf/local.conf.yaml")
	}
}

// 添加辅助函数处理本地配置加载
func loadLocalConfig(filename string) *Config {
	cfg, err := LoadSettingByFilePath(filename)
	if err != nil {
		log.Fatalf("Failed to load config file %s: %v", filename, err)
	}
	SetGlobalConf(cfg)
	return cfg
}

func LoadSettingByFilePath(filename string) (*Config, error) {
	var cfg Config
	viper.SetConfigFile(filename)
	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	// 安全地打印配置（脱敏敏感信息）
	fmt.Printf("Loaded config from: %s (env: %s, service: %s)\n",
		filename,
		cfg.ServerConf.Env,
		cfg.ServerConf.ServiceName)
	return &cfg, nil
}

func SetGlobalConf(cfg *Config) {
	GlobalConf = cfg
}

func GetServerName() string {
	return GlobalConf.ServerConf.ServiceName
}

func GetEnv() string {
	return GlobalConf.ServerConf.Env
}

func IsRunBot() bool {
	return *runBot
}

func GetMysqlMasterConf() *Mysql {
	if GlobalConf == nil {
		log.Printf("Warning: Global configuration is not initialized")
		return nil
	}
	if GlobalConf.MysqlMasterConf == nil {
		log.Printf("Warning: MySQL master configuration is not initialized")
		return nil
	}
	return GlobalConf.MysqlMasterConf
}

func GetMysqlReplicaConf() *Mysql {
	if GlobalConf == nil {
		log.Printf("Warning: Global configuration is not initialized")
		return nil
	}
	if GlobalConf.MysqlMasterConf == nil {
		log.Printf("Warning: MySQL master configuration is not initialized")
		return nil
	}
	return GlobalConf.MysqlMasterConf
}

func GetRedisConf() *Redis {
	if GlobalConf == nil {
		log.Printf("Warning: Global configuration is not initialized")
		return nil
	}
	if GlobalConf.RedisConf == nil {
		log.Printf("Warning: RedisConf master configuration is not initialized")
		return nil
	}
	return GlobalConf.RedisConf
}

func GetServeConf() *Server {
	if GlobalConf == nil {
		log.Printf("Warning: Global configuration is not initialized")
		return nil
	}
	if GlobalConf.ServerConf == nil {
		log.Printf("Warning: ServerConf master configuration is not initialized")
		return nil
	}
	return GlobalConf.ServerConf
}

// Get_MACHINE_NAME 当前实例机器名（与旧项目 conf.Get_MACHINE_NAME 一致）
func Get_MACHINE_NAME() string {
	s := GetServeConf()
	if s == nil {
		return ""
	}
	return s.MachineName
}

func GetLarkNotificationConf() *LarkNotification {
	if GlobalConf == nil {
		log.Printf("Warning: Global configuration is not initialized")
		return nil
	}
	if GlobalConf.NotificationConf == nil {
		log.Printf("Warning: NotificationConf configuration is not initialized")
		return nil
	}
	return &GlobalConf.NotificationConf.Lark
}

// PolymarketConf Polymarket 配置（市场 API base URL）
type PolymarketConf struct {
	MarketAPIBaseURL    string
	OrderBookAPIBaseURL string
}

// GetPolymarketConf 获取 Polymarket 配置，返回 nil 时使用默认 baseURL
func GetPolymarketConf() *PolymarketConf {
	if GlobalConf == nil {
		return nil
	}
	return nil
}
