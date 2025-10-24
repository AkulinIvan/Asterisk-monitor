package config

import (
    "os"
    "path/filepath"

    "asterisk-monitor/types"
    "gopkg.in/ini.v1"
)

type ConfigManager struct {
    config     *types.Config
    configPath string
}

func NewConfigManager() *ConfigManager {
    homeDir, _ := os.UserHomeDir()
    configPath := filepath.Join(homeDir, ".asterisk-monitor", "config.ini")
    
    return &ConfigManager{
        config:     &types.Config{},
        configPath: configPath,
    }
}

func (cm *ConfigManager) Load() error {
    if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
        return cm.CreateDefault()
    }
    
    cfg, err := ini.Load(cm.configPath)
    if err != nil {
        return err
    }
    
    return cfg.MapTo(cm.config)
}

func (cm *ConfigManager) Save() error {
    dir := filepath.Dir(cm.configPath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }
    
    cfg := ini.Empty()
    if err := cfg.ReflectFrom(cm.config); err != nil {
        return err
    }
    
    return cfg.SaveTo(cm.configPath)
}

func (cm *ConfigManager) CreateDefault() error {
    cm.config.Asterisk.Host = "localhost"
    cm.config.Asterisk.AMIPort = "5038"
    cm.config.Asterisk.Username = "admin"
    cm.config.Asterisk.Password = "amp111"
    
    cm.config.Monitoring.RefreshInterval = 5
    cm.config.Monitoring.EnableAlerts = true
    cm.config.Monitoring.LogRetention = 30
    
    cm.config.Security.CheckFirewall = true
    cm.config.Security.CheckPasswords = true
    cm.config.Security.CheckSSL = true
    
    return cm.Save()
}

func (cm *ConfigManager) Get() *types.Config {
    return cm.config
}

func (cm *ConfigManager) Update(newConfig *types.Config) error {
    cm.config = newConfig
    return cm.Save()
}