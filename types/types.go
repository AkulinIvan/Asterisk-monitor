package types

import "time"

type CheckResult struct {
    Name      string    `json:"name"`
    Status    string    `json:"status"` // success, warning, error
    Message   string    `json:"message"`
    Error     string    `json:"error,omitempty"`
    Timestamp time.Time `json:"timestamp"`
}

type ChannelInfo struct {
    Name        string `json:"name"`
    State       string `json:"state"`
    Duration    string `json:"duration"`
    CallerID    string `json:"callerid"`
    Application string `json:"application"`
}

type SIPPeer struct {
    Name     string `json:"name"`
    Host     string `json:"host"`
    Status   string `json:"status"`
    Latency  string `json:"latency"`
    ACL      string `json:"acl"`
}

type SystemMetrics struct {
    CPUUsage     float64 `json:"cpu_usage"`
    MemoryUsage  float64 `json:"memory_usage"`
    DiskUsage    float64 `json:"disk_usage"`
    ActiveCalls  int     `json:"active_calls"`
    TotalPeers   int     `json:"total_peers"`
    OnlinePeers  int     `json:"online_peers"`
    Uptime       string  `json:"uptime"`
    LoadAverage  string  `json:"load_average"`
    AsteriskPID  string  `json:"asterisk_pid"`
    ServiceState string  `json:"service_state"`
}

// AsteriskConfig содержит настройки подключения к Asterisk
type AsteriskConfig struct {
    Host     string `ini:"host" json:"host"`
    AMIPort  string `ini:"ami_port" json:"ami_port"`
    Username string `ini:"username" json:"username"`
    Password string `ini:"password" json:"password"`
}

// MonitoringConfig содержит настройки мониторинга
type MonitoringConfig struct {
    RefreshInterval int  `ini:"refresh_interval" json:"refresh_interval"`
    EnableAlerts    bool `ini:"enable_alerts" json:"enable_alerts"`
    LogRetention    int  `ini:"log_retention" json:"log_retention"`
}

// SecurityConfig содержит настройки безопасности
type SecurityConfig struct {
    CheckFirewall  bool `ini:"check_firewall" json:"check_firewall"`
    CheckPasswords bool `ini:"check_passwords" json:"check_passwords"`
    CheckSSL       bool `ini:"check_ssl" json:"check_ssl"`
}

type Config struct {
    Asterisk   AsteriskConfig   `ini:"asterisk" json:"asterisk"`
    Monitoring MonitoringConfig `ini:"monitoring" json:"monitoring"`
    Security   SecurityConfig   `ini:"security" json:"security"`
}