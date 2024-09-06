package sqlh

import (
	"database/sql"
	"database/sql/driver"
)

const Mock = "mock"

var _ driver.Driver = (driverPool)(nil) // required for sql.Register
var _ driver.Conn = (*mock)(nil)

// init the driverPool so that you can get a working sql.DB with sql.Open
func init() {
	d := make(driverPool)
	sql.Register(Mock, d)
}

type driverPool map[string]*mock

func (d driverPool) Open(dsn string) (driver.Conn, error) {
	if m, found := d[dsn]; found {
		return m, nil
	}

	d[dsn] = &mock{}
	return d[dsn], nil
}

func NewMock(dsn string) (*sql.DB, *mock) {
	db, _ := sql.Open(Mock, dsn)
	conn, _ := db.Driver().Open(dsn)
	return db, conn.(*mock)
}

type mock struct {
	options string
}

func (m *mock) SetOptions(o string) *mock {
	m.options = o
	return m
}
