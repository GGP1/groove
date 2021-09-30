package scan

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

var (
	// [dest type]: [field name]: field index
	mappingCache      = make(map[reflect.Type]map[string][]int)
	mu                sync.RWMutex
	_scannerInterface = reflect.TypeOf((*sql.Scanner)(nil)).Elem()
)

// Row ..
func Row(dest interface{}, rows *sql.Rows) error {
	defer rows.Close()

	value, err := destValue(dest)
	if err != nil {
		return err
	}
	if value.Kind() != reflect.Struct {
		return errors.New("dest type must be struct")
	}

	for !rows.Next() {
		return rows.Err()
	}

	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	if len(columns) == 0 {
		return nil
	}

	bType := baseType(value.Type())
	if isScannable(bType) {
		if len(columns) > 1 {
			return errors.New("scannable dest type with more than 1 column")
		}
		return rows.Scan(dest)
	}

	indices, err := columnsIndices(bType, columns)
	if err != nil {
		return err
	}

	fields := make([]interface{}, len(columns))
	for i, index := range indices {
		allocNilPointers(value, index)
		fields[i] = value.FieldByIndex(index).Addr().Interface()
	}

	return rows.Scan(fields...)
}

// Rows takes a slice of any type and scans the sql rows with it.
func Rows(dest interface{}, rows *sql.Rows) error {
	defer rows.Close()

	value, err := destValue(dest)
	if err != nil {
		return err
	}

	bType := baseType(value.Type())
	if bType.Kind() != reflect.Slice {
		return errors.New("dest must be a slice")
	}

	elem := bType.Elem()
	isPtr := elem.Kind() == reflect.Ptr
	baseElem := baseType(elem)
	if baseElem.Kind() != reflect.Struct {
		return errors.New("slice element must be a struct")
	}

	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	if len(columns) == 0 {
		return nil
	}

	indices, err := columnsIndices(baseElem, columns)
	if err != nil {
		return err
	}

	// Reuse variables
	var vPtr, v reflect.Value
	fields := make([]interface{}, len(columns))

	for rows.Next() {
		vPtr = reflect.New(baseElem)
		v = reflect.Indirect(vPtr)

		for i, index := range indices {
			allocNilPointers(v, index)
			fields[i] = v.FieldByIndex(index).Addr().Interface()
		}

		if err := rows.Scan(fields...); err != nil {
			return err
		}

		if isPtr {
			value.Set(reflect.Append(value, vPtr))
		} else {
			value.Set(reflect.Append(value, v))
		}
	}

	return rows.Err()
}

func destValue(dest interface{}) (reflect.Value, error) {
	vPtr := reflect.ValueOf(dest)
	if vPtr.Kind() != reflect.Ptr {
		return reflect.Value{}, errors.New("dest must be a pointer")
	}
	if vPtr.IsNil() {
		return reflect.Value{}, errors.New("dest mustn't be nil")
	}
	return reflect.Indirect(vPtr), nil
}

func columnsIndices(t reflect.Type, columns []string) ([][]int, error) {
	mu.Lock()
	mapping, ok := mappingCache[t]
	if !ok {
		mapping = make(map[string][]int)
		mapFields(t, mapping, nil)
		mappingCache[t] = mapping
	}
	mu.Unlock()

	indices := make([][]int, 0, len(columns))
	for _, c := range columns {
		index, ok := mapping[c]
		if !ok {
			return nil, fmt.Errorf("couldn't find a field for column %q", c)
		}

		indices = append(indices, index)
	}

	return indices, nil
}

func isScannable(t reflect.Type) bool {
	if reflect.PtrTo(t).Implements(_scannerInterface) || t.Kind() != reflect.Struct {
		return true
	}
	return false
}

func mapFields(t reflect.Type, mapping map[string][]int, parentIndices []int) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		indices := append(parentIndices, field.Index...)

		baseType := baseType(field.Type)
		kind := baseType.Kind()
		if kind == reflect.Struct {
			// if the base type is a struct, map it as well
			mapFields(baseType, mapping, indices)
		} else if kind == reflect.Slice && baseType.Elem().Kind() == reflect.Struct {
			continue
		}

		fieldName := ""
		if tag := field.Tag.Get("db"); tag != "" {
			fieldName = tag
		} else {
			fieldName = strings.ToLower(field.Name)
		}

		mapping[fieldName] = indices
	}
}

func baseType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func allocNilPointers(v reflect.Value, index []int) {
	if len(index) == 0 {
		return
	}
	if field := v.Field(index[0]); field.Kind() == reflect.Ptr {
		if field.IsNil() {
			// Field is a nil pointer, allocate a new value
			field.Set(reflect.New(field.Type().Elem()))
		}
		// Dereference the field pointer to repeat the process a level below
		allocNilPointers(reflect.Indirect(field), index[1:])
	}
}
