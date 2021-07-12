package dgraph

import (
	"context"
	"strconv"
	"strings"

	"github.com/GGP1/groove/internal/bufferpool"
	"github.com/GGP1/groove/internal/params"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"github.com/pkg/errors"
)

// Careful: functions from this file shouldn't be called
// multiple times in a sequence (except Triple), otherwise the buffer from the pool
// will get messed up.

const (
	// User dgraph type
	User dgraphType = iota
	// Event dgraph type
	Event

	// Are collisions possible?
	// Two users creating a node at the same time and hence referring to the same one,
	// in that case, assing a 4 letter random word to avoid collisions
	createSubject = "_:1"
)

type dgraphType int

// CreateNode creates a node of the given type in dgraph.
//
// CreateNode does not commit the transaction passed.
func CreateNode(ctx context.Context, tx *dgo.Txn, dType dgraphType, id string) error {
	predicate := ""
	object := ""

	switch dType {
	case User:
		predicate = "user_id"
		object = "User"
	case Event:
		predicate = "event_id"
		object = "Event"
	}

	buf := bufferpool.Get()
	// 24 is the length of the literal strings
	buf.Grow((len(createSubject) * 2) + len(predicate) + len(id) + len(object) + 24)

	// _:1 <predicate> "uuid" .
	buf.WriteString(createSubject)
	buf.WriteByte(' ')
	buf.WriteByte('<')
	buf.WriteString(predicate)
	buf.WriteByte('>')
	buf.WriteByte('"')
	buf.WriteString(id)
	buf.WriteByte('"')
	buf.WriteByte('.')
	buf.WriteByte('\n')

	// _:1 <dgraph.type> "type" .
	buf.WriteString(createSubject)
	buf.WriteString(" <dgraph.type>\"")
	buf.WriteString(object)
	buf.WriteByte('"')
	buf.WriteByte('.')
	buf.WriteByte('\n')

	mu := &api.Mutation{
		Cond:      "@if(eq(len(node), 0))",
		SetNquads: buf.Bytes(),
	}
	bufferpool.Put(buf)
	vars := map[string]string{"$id": id}
	q := `query q($id: string) {
		node as var(func: eq(` + predicate + `, $id))
	}`
	req := &api.Request{
		Vars:      vars,
		Query:     q,
		Mutations: []*api.Mutation{mu},
	}
	if _, err := tx.Do(ctx, req); err != nil {
		return errors.Wrapf(err, "creating %s node", object)
	}

	return nil
}

// EventEdgeRequest returns a new request creating an edge using upsert.
func EventEdgeRequest(eventID, predicate, userID string, set bool) *api.Request {
	vars := map[string]string{"$event_id": eventID, "$user_id": userID}
	q := `
	query q($event_id: string, $user_id: string) {
		event as var(func: eq(event_id, $event_id))
		user as var(func: eq(user_id, $user_id))
	}`
	mu := &api.Mutation{
		Cond: "@if(eq(len(event), 1) AND eq(len(user), 1))",
	}
	triple := TripleUID("uid(event)", predicate, "uid(user)")
	if set {
		mu.SetNquads = triple
	} else {
		mu.DelNquads = triple
	}

	return &api.Request{
		Vars:      vars,
		Query:     q,
		Mutations: []*api.Mutation{mu},
	}
}

// GetCount runs a query that responds with a count, it parses it and returns it.
func GetCount(ctx context.Context, dc *dgo.Dgraph, query, id string) (*uint64, error) {
	res, err := dc.NewReadOnlyTxn().QueryRDFWithVars(ctx, query, map[string]string{"$id": id})
	if err != nil {
		return nil, errors.Wrap(err, "fetching count")
	}

	return ParseCount(res.Rdf)
}

// Mutation creates a new transaction, executes the function passed and commits the results.
//
// Request made inside this function shouldn't have CommitNow set to true.
func Mutation(ctx context.Context, dc *dgo.Dgraph, f func(tx *dgo.Txn) error) error {
	tx := dc.NewTxn()
	if err := f(tx); err != nil {
		_ = tx.Discard(ctx)
		return err
	}
	if err := tx.Commit(ctx); err != nil && err != dgo.ErrFinished {
		return errors.Wrap(err, "dgraph: committing transaction")
	}
	return nil
}

// Query creates a new read-only transaction, executes the function passed and commits the results.
//
// Request made inside this function shouldn't have CommitNow set to true.
func Query(ctx context.Context, dc *dgo.Dgraph, f func(tx *dgo.Txn) (*api.Response, error)) (*api.Response, error) {
	tx := dc.NewReadOnlyTxn()
	// Since there are no mutation, commit is not needed
	return f(tx)
}

// ParseCount parses the rdf received and returns the number obtained from it.
//
// As counts are encoded like pointers, return a pointer to uint64.
func ParseCount(rdf []byte) (*uint64, error) {
	// Sample rdf:
	// <0x1> <count(predicate)> "15" .
	r := string(rdf)
	start := strings.IndexByte(r, '"') + 1
	end := strings.LastIndexByte(r, '"')
	if start == 0 || end == -1 {
		return nil, errors.Errorf("invalid rdf: %q", r)
	}
	count, err := strconv.ParseUint(r[start:end], 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "invalid number")
	}

	return &count, nil
}

// ParseCountWithMap is like ParseCount but it parses the predicates as well.
func ParseCountWithMap(rdf []byte) (map[string]*uint64, error) {
	if rdf == nil || len(rdf) == 0 {
		return nil, nil
	}

	// Sample rdf:
	// <0x1> <count(predicate)> "15" .
	lines := strings.Split(string(rdf), "\n")
	mp := make(map[string]*uint64, len(lines)-1)

	// Discard the the last line as it is empty
	for _, line := range lines[:len(lines)-1] {
		startPred := strings.IndexByte(line, '(') + 1
		endPred := strings.IndexByte(line, ')')
		if startPred == 0 || endPred == -1 {
			return nil, errors.Errorf("invalid rdf: %q", line)
		}
		pred := line[startPred:endPred]

		startCount := strings.IndexByte(line, '"') + 1
		endCount := strings.LastIndexByte(line, '"')
		if startCount == 0 || endCount == -1 {
			return nil, errors.Errorf("invalid rdf: %q", line)
		}

		count, err := strconv.ParseUint(line[startCount:endCount], 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, "invalid count")
		}
		mp[pred] = &count
	}

	return mp, nil
}

// ParseRDFUUIDs returns a slice of uuids parsed from a RDF reponse.
//
// One order of magnitude faster than using json.
func ParseRDFUUIDs(rdf []byte) []string {
	if rdf == nil || len(rdf) == 0 {
		return nil
	}
	lines := strings.Split(string(rdf), "\n")

	// Discard the first and the last line as they don't contain UUIDs
	result := make([]string, 0, len(lines)-2)
	for _, line := range lines[1 : len(lines)-1] {
		idx := strings.IndexByte(line, '"') + 1
		if idx == 0 {
			continue
		}
		// uuids are 36 chars long
		if len(line) > 35 {
			result = append(result, line[idx:idx+36])
		}
	}

	return result
}

// ParseRDFUUIDsWithMap returns a map with uuids keys parsed from a RDF reponse.
//
// One order of magnitude faster than using json.
func ParseRDFUUIDsWithMap(rdf []byte) map[string]struct{} {
	if rdf == nil || len(rdf) == 0 {
		return nil
	}
	lines := strings.Split(string(rdf), "\n")

	// Discard the first and the last line as they don't contain UUIDs
	result := make(map[string]struct{}, len(lines)-2)
	for _, line := range lines[1 : len(lines)-1] {
		idx := strings.IndexByte(line, '"') + 1
		if idx == 0 {
			continue
		}
		// uuids are 36 chars long
		if len(line) > 35 {
			result[line[idx:idx+36]] = struct{}{}
		}
	}

	return result
}

// ParseRDFWithMap works like ParseRDFReponse but it parses the predicates as well.
func ParseRDFWithMap(rdf []byte) (map[string][]string, error) {
	if rdf == nil || len(rdf) == 0 {
		return nil, nil
	}
	lines := strings.Split(string(rdf), "\n")

	predicate := ""
	// Not exactly the size of the map as some lines are predicates and
	// won't be stored but it's a good approximation. Worst case we allocate n empty spaces
	// where n is the number of lines containing predicates inside the rdf response
	mp := make(map[string][]string, len(lines)-1)
	// Discard the the last line as it's an empty line
	for _, line := range lines[:len(lines)-1] {
		// Get predicate: <0x2> <predicate> <0x1> .
		predStart := strings.IndexByte(line, '>') + 3
		predEnd := strings.IndexByte(line[predStart:], '>')
		if predStart == 2 || predEnd == -1 {
			return nil, errors.Errorf("invalid rdf: %q", line)
		}
		pred := line[predStart : predStart+predEnd]
		// Only update the predicate if it's another one
		if pred != "event_id" && pred != "user_id" {
			predicate = pred
			continue
		}

		// Look for the uuid and add it to the slice
		quoteIdx := strings.IndexByte(line, '"') + 1
		if quoteIdx == 0 {
			return nil, errors.Errorf("invalid rdf: %q", line)
		}
		mp[predicate] = append(mp[predicate], line[quoteIdx:quoteIdx+36])
	}

	return mp, nil
}

// QueryVars returns the variables used in the query depending on the parameters.
func QueryVars(id string, params params.Query) map[string]string {
	if params.LookupID != "" {
		vars := map[string]string{
			"$id":        id,
			"$lookup_id": params.LookupID,
		}
		return vars
	}

	vars := map[string]string{
		"$id":     id,
		"$cursor": params.Cursor,
		"$limit":  params.Limit,
	}
	return vars
}

// Triple builds a RDF triple.
//
// If the object is not a node uid it is enclosed by double quotes.
//
// Example:
// 	subject <predicate> object .
func Triple(subject, predicate, object string) []byte {
	literalStrings := 8
	isUID := false
	if len(object) > 3 && object[:3] == "uid" {
		isUID = true
		literalStrings -= 2
	}

	buf := bufferpool.Get()
	buf.Grow(len(subject) + len(predicate) + len(object) + literalStrings)
	buf.WriteString(subject)
	buf.WriteByte(' ')
	buf.WriteByte('<')
	buf.WriteString(predicate)
	buf.WriteByte('>')
	buf.WriteByte(' ')
	if !isUID {
		buf.WriteByte('"')
	}
	buf.WriteString(object)
	if !isUID {
		buf.WriteByte('"')
	}
	buf.WriteByte(' ')
	buf.WriteByte('.')

	triple := buf.Bytes()
	bufferpool.Put(buf)

	return triple
}

// TripleUID is like Triple but with uids.
func TripleUID(subjectUID, predicate, objectUID string) []byte {
	buf := bufferpool.Get()
	buf.Grow(len(subjectUID) + len(predicate) + len(objectUID) + 6)
	buf.WriteString(subjectUID)
	buf.WriteByte(' ')
	buf.WriteByte('<')
	buf.WriteString(predicate)
	buf.WriteByte('>')
	buf.WriteByte(' ')
	buf.WriteString(objectUID)
	buf.WriteByte(' ')
	buf.WriteByte('.')

	triple := buf.Bytes()
	bufferpool.Put(buf)

	return triple
}

// UserEdgeRequest returns a new request creating an edge using upsert.
func UserEdgeRequest(userID, predicate, targetID string, set bool) *api.Request {
	vars := map[string]string{"$user_id": userID, "$target_id": targetID}
	q := `
	query q($user_id: string, $target_id: string) {
		user as var(func: eq(user_id, $user_id))
		target as var(func: eq(user_id, $target_id))
	}`
	mu := &api.Mutation{
		Cond: "@if(eq(len(user), 1) AND eq(len(target), 1))",
	}
	triple := TripleUID("uid(user)", predicate, "uid(target)")
	if set {
		mu.SetNquads = triple
	} else {
		mu.DelNquads = triple
	}

	return &api.Request{
		Vars:      vars,
		Query:     q,
		Mutations: []*api.Mutation{mu},
		CommitNow: true,
	}
}
