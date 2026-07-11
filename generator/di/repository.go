package di

import (
	"fmt"
	"go/types"
	"strconv"
	"strings"

	"github.com/zombocoder/goboot/model"
	"github.com/zombocoder/goboot/sqlgen"
)

// renderRepositories emits the implementation struct, constructor, and methods
// for each generated repository (§27). Methods run their SQL through the
// driver-neutral db.DBTX obtained from the injected db.DBProvider; the dialect
// is applied only here, so switching drivers changes nothing else.
func renderRepositories(app *model.Application, im *imports, dialect sqlgen.Dialect) string {
	dbq := func(sym string) string { return im.qualify(dbPath, "db", sym) }

	var b strings.Builder
	for _, c := range app.Components {
		if c.Repository == nil {
			continue
		}
		iface, ok := c.ProvidedType.Underlying().(*types.Interface)
		if !ok {
			continue
		}
		b.WriteString(renderRepositoryImpl(c, iface, im, dbq, dialect))
		b.WriteString("\n")
	}
	return b.String()
}

// renderRepositoryImpl emits one repository implementation.
func renderRepositoryImpl(repo *model.Component, iface *types.Interface, im *imports, dbq func(string) string, dialect sqlgen.Dialect) string {
	implName := repo.Named.Obj().Name() + "Impl"
	methods := map[string]model.RepositoryMethod{}
	for _, m := range repo.Repository.Methods {
		methods[m.Name] = m
	}

	var b strings.Builder
	fmt.Fprintf(&b, "// %s is the generated implementation of %s.\n", implName, repo.Named.Obj().Name())
	fmt.Fprintf(&b, "type %s struct {\n\tdb %s\n}\n\n", implName, dbq("DBProvider"))
	fmt.Fprintf(&b, "// New%s builds the repository.\n", implName)
	fmt.Fprintf(&b, "func New%s(db %s) *%s {\n\treturn &%s{db: db}\n}\n\n", implName, dbq("DBProvider"), implName, implName)

	for i := 0; i < iface.NumMethods(); i++ {
		method := iface.Method(i)
		m, ok := methods[method.Name()]
		if !ok {
			continue
		}
		b.WriteString(renderRepositoryMethod(implName, m, im, dbq, dialect))
		b.WriteString("\n")
	}
	return b.String()
}

// renderRepositoryMethod emits a single query or exec method.
func renderRepositoryMethod(implName string, m model.RepositoryMethod, im *imports, dbq func(string) string, dialect sqlgen.Dialect) string {
	sig := m.Signature
	params, argNames := renderParamList(sig, im)
	results := renderResultList(sig, im, false)
	ctxVar := "a0"
	if len(argNames) > 0 {
		ctxVar = argNames[0]
	}

	compiled := sqlgen.Compile(m.RawSQL, dialect)
	sqlLit := sqlLiteral(compiled.SQL)

	var b strings.Builder
	fmt.Fprintf(&b, "func (r *%s) %s(%s) %s {\n", implName, m.Name, params, results)
	dbExpr := fmt.Sprintf("r.db.DB(%s)", ctxVar)

	switch m.Kind {
	case model.QueryBatch:
		b.WriteString(renderBatchBody(m, dbExpr, ctxVar, sqlLit, compiled.Params))
	case model.QueryExec:
		b.WriteString(renderExecBody(m, dbExpr, ctxVar, sqlLit, sqlArgs(compiled.Params, sig)))
	default:
		b.WriteString(renderQueryBody(m, dbExpr, ctxVar, sqlLit, sqlArgs(compiled.Params, sig), im))
	}
	b.WriteString("}\n")
	return b.String()
}

// renderBatchBody emits an @Batch method body that runs its statement once per
// element of the iterated slice parameter (§27.3). SQL parameters based on the
// slice bind to the current element; other parameters bind directly.
func renderBatchBody(m model.RepositoryMethod, dbExpr, ctxVar, sqlLit string, refs []string) string {
	const loopVar = "item"
	slice := "a" + strconv.Itoa(m.Batch.ParamIndex)
	exec := fmt.Sprintf("%s.ExecContext(%s, %s%s)", dbExpr, ctxVar, sqlLit,
		commaPrefixed(batchSQLArgs(refs, m.Signature, m.Batch, loopVar)))

	var b strings.Builder
	if m.Return.RowsAffected {
		b.WriteString("\tvar affected int64\n")
		fmt.Fprintf(&b, "\tfor _, %s := range %s {\n", loopVar, slice)
		fmt.Fprintf(&b, "\t\tres, err := %s\n", exec)
		b.WriteString("\t\tif err != nil {\n\t\t\treturn affected, err\n\t\t}\n")
		b.WriteString("\t\tn, err := res.RowsAffected()\n")
		b.WriteString("\t\tif err != nil {\n\t\t\treturn affected, err\n\t\t}\n")
		b.WriteString("\t\taffected += n\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn affected, nil\n")
		return b.String()
	}
	fmt.Fprintf(&b, "\tfor _, %s := range %s {\n", loopVar, slice)
	fmt.Fprintf(&b, "\t\tif _, err := %s; err != nil {\n\t\t\treturn err\n\t\t}\n", exec)
	b.WriteString("\t}\n")
	b.WriteString("\treturn nil\n")
	return b.String()
}

// batchSQLArgs maps compiled SQL parameter references for an @Batch method:
// references based on the iterated slice bind to the loop variable, others to
// the method's renamed arguments.
func batchSQLArgs(refs []string, sig *types.Signature, batch *model.BatchInfo, loopVar string) string {
	byOrig := map[string]string{}
	for i := 0; i < sig.Params().Len(); i++ {
		if n := sig.Params().At(i).Name(); n != "" {
			byOrig[n] = "a" + strconv.Itoa(i)
		}
	}
	args := make([]string, len(refs))
	for i, ref := range refs {
		base, field := ref, ""
		if dot := strings.IndexByte(ref, '.'); dot >= 0 {
			base, field = ref[:dot], ref[dot:]
		}
		switch {
		case base == batch.ParamName:
			args[i] = loopVar + field
		default:
			renamed, ok := byOrig[base]
			if !ok {
				renamed = base
			}
			args[i] = renamed + field
		}
	}
	return strings.Join(args, ", ")
}

// renderExecBody emits an @Exec method body.
func renderExecBody(m model.RepositoryMethod, dbExpr, ctxVar, sqlLit, args string) string {
	exec := fmt.Sprintf("%s.ExecContext(%s, %s%s)", dbExpr, ctxVar, sqlLit, commaPrefixed(args))
	if m.Return.RowsAffected {
		var b strings.Builder
		fmt.Fprintf(&b, "\tres, err := %s\n", exec)
		b.WriteString("\tif err != nil {\n\t\treturn 0, err\n\t}\n")
		b.WriteString("\treturn res.RowsAffected()\n")
		return b.String()
	}
	return fmt.Sprintf("\t_, err := %s\n\treturn err\n", exec)
}

// renderQueryBody emits an @Query method body for a single row, slice, or scalar.
func renderQueryBody(m model.RepositoryMethod, dbExpr, ctxVar, sqlLit, args string, im *imports) string {
	if m.Return.Multi {
		return renderSliceQuery(m, dbExpr, ctxVar, sqlLit, args, im)
	}
	return renderSingleQuery(m, dbExpr, ctxVar, sqlLit, args, im)
}

// renderSingleQuery emits a single-row query (struct pointer or scalar).
func renderSingleQuery(m model.RepositoryMethod, dbExpr, ctxVar, sqlLit, args string, im *imports) string {
	queryRow := fmt.Sprintf("%s.QueryRowContext(%s, %s%s)", dbExpr, ctxVar, sqlLit, commaPrefixed(args))
	var b strings.Builder
	if m.Return.Scalar {
		typ := renderType(m.Return.Elem, im)
		fmt.Fprintf(&b, "\tvar v %s\n", typ)
		fmt.Fprintf(&b, "\tif err := %s.Scan(&v); err != nil {\n\t\treturn %s, err\n\t}\n", queryRow, zeroValue(m.Return.Elem))
		b.WriteString("\treturn v, nil\n")
		return b.String()
	}
	// Struct entity.
	entity := renderType(m.Return.Elem, im)
	dests := scanDests(m.Return.Elem)
	fmt.Fprintf(&b, "\tvar e %s\n", entity)
	fmt.Fprintf(&b, "\tif err := %s.Scan(%s); err != nil {\n\t\treturn nil, err\n\t}\n", queryRow, dests)
	b.WriteString("\treturn &e, nil\n")
	return b.String()
}

// renderSliceQuery emits a multi-row query returning a slice.
func renderSliceQuery(m model.RepositoryMethod, dbExpr, ctxVar, sqlLit, args string, im *imports) string {
	query := fmt.Sprintf("%s.QueryContext(%s, %s%s)", dbExpr, ctxVar, sqlLit, commaPrefixed(args))
	elem := renderType(m.Return.Elem, im)
	sliceType := "[]" + elem
	if m.Return.Pointer {
		sliceType = "[]*" + elem
	}

	var b strings.Builder
	fmt.Fprintf(&b, "\trows, err := %s\n", query)
	b.WriteString("\tif err != nil {\n\t\treturn nil, err\n\t}\n")
	b.WriteString("\tdefer rows.Close()\n")
	fmt.Fprintf(&b, "\tvar out %s\n", sliceType)
	b.WriteString("\tfor rows.Next() {\n")
	if m.Return.Scalar {
		fmt.Fprintf(&b, "\t\tvar v %s\n", elem)
		b.WriteString("\t\tif err := rows.Scan(&v); err != nil {\n\t\t\treturn nil, err\n\t\t}\n")
		b.WriteString("\t\tout = append(out, v)\n")
	} else {
		fmt.Fprintf(&b, "\t\tvar e %s\n", elem)
		fmt.Fprintf(&b, "\t\tif err := rows.Scan(%s); err != nil {\n\t\t\treturn nil, err\n\t\t}\n", scanDests(m.Return.Elem))
		if m.Return.Pointer {
			b.WriteString("\t\tout = append(out, &e)\n")
		} else {
			b.WriteString("\t\tout = append(out, e)\n")
		}
	}
	b.WriteString("\t}\n")
	b.WriteString("\treturn out, rows.Err()\n")
	return b.String()
}

// scanDests renders the &e.Field destination list for a struct entity's exported
// fields, in declaration order (§27.8 option 1).
func scanDests(entity types.Type) string {
	st, ok := entity.Underlying().(*types.Struct)
	if !ok {
		return ""
	}
	var dests []string
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		if f.Exported() {
			dests = append(dests, "&e."+f.Name())
		}
	}
	return strings.Join(dests, ", ")
}

// sqlArgs maps compiled SQL parameter references to the method's renamed
// arguments, e.g. the reference "id" to "a1" and "user.ID" to "a1.ID".
func sqlArgs(refs []string, sig *types.Signature) string {
	byOrig := map[string]string{}
	for i := 0; i < sig.Params().Len(); i++ {
		if n := sig.Params().At(i).Name(); n != "" {
			byOrig[n] = "a" + strconv.Itoa(i)
		}
	}
	args := make([]string, len(refs))
	for i, ref := range refs {
		base, field := ref, ""
		if dot := strings.IndexByte(ref, '.'); dot >= 0 {
			base, field = ref[:dot], ref[dot:]
		}
		renamed, ok := byOrig[base]
		if !ok {
			renamed = base // discovery reported this; keep the ref so it surfaces
		}
		args[i] = renamed + field
	}
	return strings.Join(args, ", ")
}

// commaPrefixed prepends ", " to a non-empty argument list.
func commaPrefixed(args string) string {
	if args == "" {
		return ""
	}
	return ", " + args
}

// sqlLiteral renders SQL as a Go string literal, preferring a raw string for
// readability when the SQL contains no backtick.
func sqlLiteral(sql string) string {
	if !strings.Contains(sql, "`") {
		return "`" + sql + "`"
	}
	return strconv.Quote(sql)
}

// zeroValue renders the zero literal for a scalar type used in an error return.
func zeroValue(t types.Type) string {
	b, ok := t.Underlying().(*types.Basic)
	if !ok {
		return "0"
	}
	switch {
	case b.Info()&types.IsString != 0:
		return `""`
	case b.Info()&types.IsBoolean != 0:
		return "false"
	default:
		return "0"
	}
}
