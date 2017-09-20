package dsc

import (
	"fmt"
	"github.com/viant/toolbox"
	"path"
	"strings"
)

const defaultTableSQL = "SELECT table_name AS name FROM  information_schema.tables WHERE table_schema = ?"
const defaultSequenceSQL = "SELECT auto_increment FROM information_schema.tables WHERE table_name = '%v' AND table_schema = DATABASE()"
const defaultKeySQL = "SELECT column_name AS name FROM information_schema.key_column_usage WHERE table_name = '%v' AND table_schema = '%v' AND constraint_name='PRIMARY'"

const defaultSchemaSQL = "SELECT DATABASE() AS name"
const defaultAllSchemaSQL = "SELECT schema_name AS name FROM  information_schema.schemata"

const mysqlDisableForeignCheck = "SET FOREIGN_KEY_CHECKS=0"
const mysqlEnableForeignCheck = "SET FOREIGN_KEY_CHECKS=1"

const schemaSQL = "SELECT current_schema() AS name"

const sqlLightTableSQL = "SELECT name FROM SQLITE_MASTER WHERE type='table' AND name NOT IN('sqlite_sequence') AND LENGTH(?) > 0"
const sqlLightSequenceSQL = "SELECT COALESCE(MAX(name), 0) + 1   FROM (SELECT seq AS name FROM SQLITE_SEQUENCE WHERE name = '%v')"
const sqlLightSchemaSQL = "PRAGMA database_list"

const pgSequenceSQL = "SELECT currval(%v) + 1"

const oraTableSQL = "SELECT table_name AS name  FROM all_tables WHERE owner = ?"
const oraSchemaSQL = "SELECT sys_context( 'userenv', 'current_schema' ) AS name FROM dual"
const oraSequenceSQL = "SELECT %v.nextval AS name from dual"
const oraAllSchemaSQL = "SELECT schema_name AS name FROM all_tables GROUP BY 1"

const msSchemaSQL = "SELECT SCHEMA_NAME() AS name"
const msSequenceSQL = "SELECT current_value FROM sys.sequences WHERE  name = '%v'"

type nameRecord struct {
	Name string `column:"name"`
}

type sqlDatastoreDialect struct {
	tablesSQL              string
	sequenceSQL            string
	schemaSQL              string
	allSchemaSQL           string
	keySQL                 string
	disableForeignKeyCheck string
	enableForeignKeyCheck  string
	schemaResultsetIndex   int
}

//CanDropDatastore returns true if this dialect can create datastore
func (d sqlDatastoreDialect) CanCreateDatastore(manager Manager) bool {
	return true
}

//CanDropDatastore returns true if this dialect can drop datastore
func (d sqlDatastoreDialect) CanDropDatastore(manager Manager) bool {
	return true
}

//CreateDatastore create a new datastore (database/schema), it takes manager and target datastore
func (d sqlDatastoreDialect) CreateDatastore(manager Manager, datastore string) error {
	_, err := manager.Execute("CREATE DATABASE " + datastore)
	return err
}

//DropTable drops a datastore (database/schema), it takes manager and datastore to be droped
func (d sqlDatastoreDialect) DropDatastore(manager Manager, datastore string) error {
	_, err := manager.Execute("DROP DATABASE " + datastore)
	return err
}

//DropTable drops a table in datastore managed by manager.
func (d sqlDatastoreDialect) DropTable(manager Manager, datastore string, table string) error {
	_, err := manager.Execute("DROP TABLE " + table)
	return err
}

//CreateTable creates table on in datastore managed by manager.
func (d sqlDatastoreDialect) CreateTable(manager Manager, datastore string, table string, specification string) error {
	_, err := manager.Execute("CREATE TABLE " + table + "(" + specification + ")")
	return err
}

//GetTables return tables names for passed in datastore managed by manager.
func (d sqlDatastoreDialect) GetTables(manager Manager, datastore string) ([]string, error) {
	var rows = make([]nameRecord, 0)
	err := manager.ReadAll(&rows, d.tablesSQL, []interface{}{datastore}, nil)
	if err != nil {
		return nil, err
	}
	var result = make([]string, 0)
	for _, row := range rows {
		if len(row.Name) > 0 {
			result = append(result, row.Name)
		}
	}
	return result, nil
}

func normalizeName(name string) string {
	if !strings.Contains(name, "/") && !strings.Contains(name, "\\") {
		return name
	}
	_, file := path.Split(name)
	return file
}

//GetKeyName returns key name
func (d sqlDatastoreDialect) GetKeyName(manager Manager, datastore, table string) string {
	if d.keySQL == "" {
		return ""
	}
	var records = make([]map[string]interface{}, 0)
	err := manager.ReadAll(&records, fmt.Sprintf(d.keySQL, table, datastore), []interface{}{}, nil)
	if err != nil {
		return ""
	}
	var result = make([]string, 0)
	for _, item := range records {
		result = append(result, toolbox.AsString(item["name"]))
	}
	return strings.Join(result, ",")
}

//GetDatastores returns name of datastores, takes  manager as parameter
func (d sqlDatastoreDialect) GetDatastores(manager Manager) ([]string, error) {
	var rows = make([][]interface{}, 0)
	err := manager.ReadAll(&rows, d.allSchemaSQL, nil, nil)
	if err != nil {
		return nil, err
	}
	var result = make([]string, 0)
	for _, row := range rows {
		result = append(result, normalizeName(toolbox.AsString(row[d.schemaResultsetIndex])))
	}
	return result, nil
}

//GetCurrentDatastore returns name of current schema
func (d sqlDatastoreDialect) GetCurrentDatastore(manager Manager) (string, error) {
	var result = make([]interface{}, 0)
	success, err := manager.ReadSingle(&result, d.schemaSQL, nil, nil)
	if err != nil || !success {
		return "", err
	}
	return normalizeName(toolbox.AsString(result[d.schemaResultsetIndex])), nil

}

//GetSequence returns sequence value or error for passed in manager and table/sequence
func (d sqlDatastoreDialect) GetSequence(manager Manager, name string) (int64, error) {
	var result = make([]int64, 0)
	success, err := manager.ReadSingle(&result, fmt.Sprintf(d.sequenceSQL, name), []interface{}{}, nil)
	if err != nil || !success {
		return 0, err
	}
	return result[0], nil
}

//DisableForeignKeyCheck disables fk check
func (d sqlDatastoreDialect) DisableForeignKeyCheck(manager Manager) error {
	if d.disableForeignKeyCheck == "" {
		return nil
	}
	_, err := manager.Execute(d.disableForeignKeyCheck)
	return err
}

//EnableForeignKeyCheck disables fk check
func (d sqlDatastoreDialect) EnableForeignKeyCheck(manager Manager) error {
	if d.enableForeignKeyCheck == "" {
		return nil
	}
	_, err := manager.Execute(d.enableForeignKeyCheck)
	return err
}

//CanPersistBatch return true if datastore can persist in batch
func (d sqlDatastoreDialect) CanPersistBatch() bool {
	return false
}

//NewSQLDatastoreDialect creates a new default sql dialect
func NewSQLDatastoreDialect(tablesSQL, sequenceSQL, schemaSQL, allSchemaSQL, keySQL, disableForeignKeyCheck, enableForeignKeyCheck string, schmeaIndex int) DatastoreDialect {
	return &sqlDatastoreDialect{tablesSQL, sequenceSQL, schemaSQL, allSchemaSQL, keySQL, disableForeignKeyCheck, enableForeignKeyCheck, schmeaIndex}
}

type mySQLDialect struct {
	DatastoreDialect
}

func newMySQLDialect() mySQLDialect {
	return mySQLDialect{DatastoreDialect: NewSQLDatastoreDialect(defaultTableSQL, defaultSequenceSQL, defaultSchemaSQL, defaultAllSchemaSQL, defaultKeySQL, mysqlDisableForeignCheck, mysqlEnableForeignCheck, 0)}
}

type sqlLiteDialect struct {
	DatastoreDialect
}

//CreateDatastore create a new datastore (database/schema), it takes manager and target datastore
func (d sqlLiteDialect) CreateDatastore(manager Manager, datastore string) error {
	return nil
}

func (d sqlLiteDialect) DropDatastore(manager Manager, datastore string) error {
	tables, err := d.GetTables(manager, datastore)
	if err != nil {
		return err
	}
	for _, table := range tables {
		err := d.DropTable(manager, datastore, table)
		if err != nil {
			return err
		}
	}
	return err
}

func newSQLLiteDialect() *sqlLiteDialect {
	return &sqlLiteDialect{DatastoreDialect: NewSQLDatastoreDialect(sqlLightTableSQL, sqlLightSequenceSQL, sqlLightSchemaSQL, sqlLightSchemaSQL, "", "", "", 2)}
}

type pgDialect struct {
	DatastoreDialect
}

func newPgDialect() *pgDialect {
	return &pgDialect{DatastoreDialect: NewSQLDatastoreDialect(sqlLightTableSQL, pgSequenceSQL, schemaSQL, defaultAllSchemaSQL, "", "", "", 0)}
}

type oraDialect struct {
	DatastoreDialect
}

//CreateDatastore create a new datastore (database/schema), it takes manager and target datastore
func (d oraDialect) CreateDatastore(manager Manager, datastore string) error {
	_, err := manager.Execute("CREATE SCHEMA IF NOT EXISTS " + datastore)
	return err
}

//DropTable drops a datastore (database/schema), it takes manager and datastore to be droped
func (d oraDialect) DropDatastore(manager Manager, datastore string) error {
	_, err := manager.Execute("DROP SCHEMA " + datastore)
	return err
}

func newOraDialect() *oraDialect {
	return &oraDialect{DatastoreDialect: NewSQLDatastoreDialect(oraTableSQL, oraSequenceSQL, oraSchemaSQL, oraAllSchemaSQL, "", "", "", 0)}
}

type msSQLDialect struct {
	DatastoreDialect
}

func newMsSQLDialect() *msSQLDialect {
	return &msSQLDialect{DatastoreDialect: NewSQLDatastoreDialect(defaultTableSQL, msSequenceSQL, msSchemaSQL, defaultAllSchemaSQL, "", "", "", 0)}
}
