package entity

import "time"

// Account 下单账户与全部 API 凭据
type Account struct {
	ID                      uint      `gorm:"primaryKey;column:id;type:bigint unsigned;not null;autoIncrement"`
	Platform                string    `gorm:"column:platform;type:varchar(32);not null"`
	Name                    string    `gorm:"column:name;type:varchar(64);not null;default:''"`
	AuthMode                string    `gorm:"column:auth_mode;type:varchar(32);not null;default:CLOB"`
	Status                  string    `gorm:"column:status;type:varchar(20);not null;default:ACTIVE"`
	APIKey                  *string   `gorm:"column:api_key;type:varchar(256)"`
	APISecretEnc            *string   `gorm:"column:api_secret_enc;type:varchar(512)"`
	APIPassphraseEnc        *string   `gorm:"column:api_passphrase_enc;type:varchar(512)"`
	BuilderAPIKey           *string   `gorm:"column:builder_api_key;type:varchar(128)"`
	BuilderAPISecretEnc     *string   `gorm:"column:builder_api_secret_enc;type:varchar(512)"`
	BuilderAPIPassphraseEnc *string   `gorm:"column:builder_api_passphrase_enc;type:varchar(512)"`
	KeyID                   *string   `gorm:"column:key_id;type:varchar(256)"`
	EoaAddress              *string   `gorm:"column:eoa_address;type:varchar(128)"`
	ProxyAddress            *string   `gorm:"column:proxy_address;type:varchar(128)"`
	PrivateKeyEnc           *string   `gorm:"column:private_key_enc;type:varchar(1024)"`
	SignatureType           *int      `gorm:"column:signature_type;type:int"`
	UseMagicAccount         int       `gorm:"column:use_magic_account;type:tinyint(1);not null;default:0"`
	ForegateUserID          *int      `gorm:"column:foregate_user_id;type:int"`
	AccountType             *string   `gorm:"column:account_type;type:varchar(16)"`
	LoginName               *string   `gorm:"column:login_name;type:varchar(128)"`
	LoginPasswordEnc        *string   `gorm:"column:login_password_enc;type:varchar(512)"`
	FgTokenEnc              *string   `gorm:"column:fg_token_enc;type:varchar(512)"`
	OrderMaxPendingTime     *int      `gorm:"column:order_max_pending_time;type:int"`
	Config                  *string   `gorm:"column:config;type:text"`
	Remark                  string    `gorm:"column:remark;type:varchar(500);not null;default:''"`
	CreatedAt               time.Time `gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt               time.Time `gorm:"column:updated_at;type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

func (Account) TableName() string {
	return "account"
}
