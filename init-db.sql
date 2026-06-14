-- CMDB 资产管理系统初始化 SQL
-- 注意：此文件仅在 PostgreSQL Docker 容器首次启动时执行
-- 后续 schema 变更由 GORM AutoMigrate 管理

CREATE TABLE IF NOT EXISTS users (
  id SERIAL PRIMARY KEY,
  username VARCHAR(50) UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  role VARCHAR(20) DEFAULT 'user',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS idc_info (
  id SERIAL PRIMARY KEY,
  zbx_id VARCHAR(50) NOT NULL,
  ipmi_ip VARCHAR(16) UNIQUE NOT NULL,
  idc_code VARCHAR(10) NOT NULL,
  idc_name VARCHAR(50) NOT NULL,
  ssh_ip VARCHAR(16) NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS machine_info (
  id SERIAL PRIMARY KEY,
  zbx_id VARCHAR(50) NOT NULL,
  ipmi_ip VARCHAR(16) NOT NULL,
  system_type VARCHAR(20) NOT NULL,
  manufacturer VARCHAR(20) NOT NULL,
  server_sn VARCHAR(50) NOT NULL,
  system_disk VARCHAR(20) NOT NULL,
  ssd_count VARCHAR(20) NOT NULL,
  hdd_count VARCHAR(20) NOT NULL,
  memory_count VARCHAR(20) NOT NULL,
  cpu_info TEXT NOT NULL,
  server_height VARCHAR(10) NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS business_info (
  id SERIAL PRIMARY KEY,
  zbx_id VARCHAR(50) NOT NULL,
  ipmi_ip VARCHAR(16) NOT NULL,
  business_name VARCHAR(100) NOT NULL,
  business_id VARCHAR(50) NOT NULL,
  old_business_name VARCHAR(100) NOT NULL,
  old_business_id VARCHAR(50) NOT NULL,
  business_speed SMALLINT NOT NULL DEFAULT 0,
  old_business_speed SMALLINT NOT NULL DEFAULT 0,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- network_info: 注意实际表中的 ipv4_ip / ipv6_ip 允许 NULL
-- zbx_id / ipmi_ip / idc_code 允许空字符串（未关联到具体机器的IP记录）
CREATE TABLE IF NOT EXISTS network_info (
  id SERIAL PRIMARY KEY,
  ipmi_ip VARCHAR(16),
  ipv4_ip VARCHAR(20),
  zbx_id VARCHAR(50),
  mac_address VARCHAR(17),
  eth_name VARCHAR(15),
  idc_code VARCHAR(10),
  net_type VARCHAR(20),
  vlan VARCHAR(9),
  ipv4_gateway VARCHAR(20),
  ipv6_ip VARCHAR(50),
  ipv6_gateway VARCHAR(50),
  ip_speed SMALLINT,
  ip_status VARCHAR(10),
  ip_notes VARCHAR(255),
  segment_notes VARCHAR(255),
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS version_info (
  version_num VARCHAR(20) NOT NULL
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_machine_info_ipmi_ip ON machine_info(ipmi_ip);
CREATE INDEX IF NOT EXISTS idx_business_info_ipmi_ip ON business_info(ipmi_ip);
CREATE INDEX IF NOT EXISTS idx_idc_info_ipmi_ip ON idc_info(ipmi_ip);
CREATE INDEX IF NOT EXISTS idx_idc_info_idc_code ON idc_info(idc_code);
CREATE INDEX IF NOT EXISTS idx_network_info_zbx_id ON network_info(zbx_id);
CREATE INDEX IF NOT EXISTS idx_network_info_ipmi_ip ON network_info(ipmi_ip);
CREATE INDEX IF NOT EXISTS idx_network_info_idc_code ON network_info(idc_code);
CREATE INDEX IF NOT EXISTS idx_network_info_ipv4_ip ON network_info(ipv4_ip);
CREATE UNIQUE INDEX IF NOT EXISTS idx_network_info_ipv4_unique ON network_info(ipv4_ip) WHERE ipv4_ip IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_network_info_ipv6_unique ON network_info(ipv6_ip) WHERE ipv6_ip IS NOT NULL;

-- 初始化用户（密码由 Go 程序在 seedUsers() 中通过 bcrypt 设置）
-- 此处仅作为占位，实际密码哈希由应用层管理
INSERT INTO users (username, password_hash, role) VALUES
  ('admin', '$2a$10$placeholder_admin_hash_to_be_updated_by_app', 'admin'),
  ('bdkj', '$2a$10$placeholder_user_hash_to_be_updated_by_app', 'user')
ON CONFLICT (username) DO NOTHING;
