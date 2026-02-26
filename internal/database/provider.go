/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package database

import "fmt"

// Hướng dẫn đọc nhanh:
// - Nếu chưa rõ nơi dùng provider, xem internal/reconciler/database.go.
// - Nếu chưa rõ cấu hình DB trong spec, xem api/v1/musicservice_types.go.

// Provider trừu tượng hóa cấu hình theo từng loại cơ sở dữ liệu
type Provider interface {
	Name() string
	DefaultImage() string
	DefaultPort() int32
	DefaultRootPassword() string
	DefaultStorageSize() string
	BuildInitReplicationScript(masterHost, password string) string
}

// MariaDBProvider triển khai Provider cho MariaDB
type MariaDBProvider struct{}

func (p *MariaDBProvider) Name() string {
	return "mariadb"
}

func (p *MariaDBProvider) DefaultImage() string {
	return "mariadb:10.11"
}

func (p *MariaDBProvider) DefaultPort() int32 {
	return 3306
}

func (p *MariaDBProvider) DefaultRootPassword() string {
	return "rootpass"
}

func (p *MariaDBProvider) DefaultStorageSize() string {
	return "10Gi"
}

func (p *MariaDBProvider) BuildInitReplicationScript(masterHost, password string) string {
	return fmt.Sprintf(`
#!/bin/bash
set -e
echo "Waiting for master to be ready..."
until mysql -h %s -uroot -p%s -e "SELECT 1" > /dev/null 2>&1; do
  sleep 2
done
echo "Master is ready, configuring replication..."
`, masterHost, password)
}

// PostgreSQLProvider triển khai Provider cho PostgreSQL
type PostgreSQLProvider struct{}

func (p *PostgreSQLProvider) Name() string {
	return "postgresql"
}

func (p *PostgreSQLProvider) DefaultImage() string {
	return "postgres:15"
}

func (p *PostgreSQLProvider) DefaultPort() int32 {
	return 5432
}

func (p *PostgreSQLProvider) DefaultRootPassword() string {
	return "postgres"
}

func (p *PostgreSQLProvider) DefaultStorageSize() string {
	return "10Gi"
}

func (p *PostgreSQLProvider) BuildInitReplicationScript(masterHost, password string) string {
	return fmt.Sprintf(`
#!/bin/bash
set -e
echo "Waiting for master to be ready..."
until pg_isready -h %s -U postgres > /dev/null 2>&1; do
  sleep 2
done
echo "Master is ready, configuring replication..."
`, masterHost)
}

// MySQLProvider triển khai Provider cho MySQL
type MySQLProvider struct{}

func (p *MySQLProvider) Name() string {
	return "mysql"
}

func (p *MySQLProvider) DefaultImage() string {
	return "mysql:8.0"
}

func (p *MySQLProvider) DefaultPort() int32 {
	return 3306
}

func (p *MySQLProvider) DefaultRootPassword() string {
	return "rootpass"
}

func (p *MySQLProvider) DefaultStorageSize() string {
	return "10Gi"
}

func (p *MySQLProvider) BuildInitReplicationScript(masterHost, password string) string {
	return fmt.Sprintf(`
#!/bin/bash
set -e
echo "Waiting for master to be ready..."
until mysql -h %s -uroot -p%s -e "SELECT 1" > /dev/null 2>&1; do
  sleep 2
done
echo "Master is ready, configuring replication..."
`, masterHost, password)
}

// Registry cho các provider cơ sở dữ liệu
var providers = map[string]Provider{
	"mariadb":    &MariaDBProvider{},
	"postgresql": &PostgreSQLProvider{},
	"mysql":      &MySQLProvider{},
}

// GetProvider trả về provider cho loại cơ sở dữ liệu đã cho
func GetProvider(dbType string) Provider {
	if p, ok := providers[dbType]; ok {
		return p
	}
	// Mặc định dùng MariaDB nếu không tìm thấy
	return providers["mariadb"]
}

// RegisterProvider đăng ký một provider tùy chỉnh
func RegisterProvider(name string, provider Provider) {
	providers[name] = provider
}
