package analyzer

import (
	"fmt"
	"testing"

	"gopkg.in/src-d/go-mysql-server.v0/mem"
	"gopkg.in/src-d/go-mysql-server.v0/sql"
	"gopkg.in/src-d/go-mysql-server.v0/sql/expression"
	"gopkg.in/src-d/go-mysql-server.v0/sql/plan"

	"github.com/stretchr/testify/require"
)

func TestAnalyzer_Analyze(t *testing.T) {
	require := require.New(t)

	table := mem.NewTable("mytable", sql.Schema{{Name: "i", Type: sql.Int32, Source: "mytable"}})
	table2 := mem.NewTable("mytable2", sql.Schema{{Name: "i2", Type: sql.Int32, Source: "mytable2"}})
	db := mem.NewDatabase("mydb")

	memDb, ok := db.(*mem.Database)
	require.True(ok)

	memDb.AddTable("mytable", table)
	memDb.AddTable("mytable2", table2)

	catalog := &sql.Catalog{Databases: []sql.Database{db}}
	a := New(catalog)
	a.CurrentDatabase = "mydb"

	var notAnalyzed sql.Node = plan.NewUnresolvedTable("mytable")
	analyzed, err := a.Analyze(notAnalyzed)
	require.NoError(err)
	require.Equal(table, analyzed)

	notAnalyzed = plan.NewUnresolvedTable("nonexistant")
	analyzed, err = a.Analyze(notAnalyzed)
	require.Error(err)
	require.Nil(analyzed)

	analyzed, err = a.Analyze(table)
	require.NoError(err)
	require.Equal(table, analyzed)

	notAnalyzed = plan.NewProject(
		[]sql.Expression{expression.NewUnresolvedColumn("o")},
		plan.NewUnresolvedTable("mytable"),
	)
	_, err = a.Analyze(notAnalyzed)
	require.Error(err)

	notAnalyzed = plan.NewProject(
		[]sql.Expression{expression.NewUnresolvedColumn("i")},
		plan.NewUnresolvedTable("mytable"),
	)
	analyzed, err = a.Analyze(notAnalyzed)
	var expected sql.Node = plan.NewProject(
		[]sql.Expression{expression.NewGetFieldWithTable(0, sql.Int32, "mytable", "i", false)},
		table,
	)
	require.NoError(err)
	require.Equal(expected, analyzed)

	notAnalyzed = plan.NewDescribe(
		plan.NewUnresolvedTable("mytable"),
	)
	analyzed, err = a.Analyze(notAnalyzed)
	expected = plan.NewDescribe(table)
	require.NoError(err)
	require.Equal(expected, analyzed)

	notAnalyzed = plan.NewProject(
		[]sql.Expression{expression.NewStar()},
		plan.NewUnresolvedTable("mytable"),
	)
	analyzed, err = a.Analyze(notAnalyzed)
	expected = plan.NewProject(
		[]sql.Expression{expression.NewGetField(0, sql.Int32, "i", false)},
		table,
	)
	require.NoError(err)
	require.Equal(expected, analyzed)

	notAnalyzed = plan.NewProject(
		[]sql.Expression{expression.NewStar()},
		plan.NewProject(
			[]sql.Expression{expression.NewStar()},
			plan.NewUnresolvedTable("mytable"),
		),
	)
	analyzed, err = a.Analyze(notAnalyzed)
	expected = plan.NewProject(
		[]sql.Expression{expression.NewGetField(0, sql.Int32, "i", false)},
		plan.NewProject(
			[]sql.Expression{expression.NewGetField(0, sql.Int32, "i", false)},
			table,
		),
	)
	require.NoError(err)
	require.Equal(expected, analyzed)

	notAnalyzed = plan.NewProject(
		[]sql.Expression{
			expression.NewAlias(
				expression.NewUnresolvedColumn("i"),
				"foo",
			),
		},
		plan.NewUnresolvedTable("mytable"),
	)
	analyzed, err = a.Analyze(notAnalyzed)
	expected = plan.NewProject(
		[]sql.Expression{
			expression.NewAlias(
				expression.NewGetFieldWithTable(0, sql.Int32, "mytable", "i", false),
				"foo",
			),
		},
		table,
	)
	require.NoError(err)
	require.Equal(expected, analyzed)

	notAnalyzed = plan.NewProject(
		[]sql.Expression{expression.NewUnresolvedColumn("i")},
		plan.NewFilter(
			expression.NewEquals(
				expression.NewUnresolvedColumn("i"),
				expression.NewLiteral(int32(1), sql.Int32),
			),
			plan.NewUnresolvedTable("mytable"),
		),
	)
	analyzed, err = a.Analyze(notAnalyzed)
	expected = plan.NewProject(
		[]sql.Expression{
			expression.NewGetFieldWithTable(0, sql.Int32, "mytable", "i", false),
		},
		plan.NewFilter(
			expression.NewEquals(
				expression.NewGetFieldWithTable(0, sql.Int32, "mytable", "i", false),
				expression.NewLiteral(int32(1), sql.Int32),
			),
			table,
		),
	)
	require.NoError(err)
	require.Equal(expected, analyzed)

	notAnalyzed = plan.NewProject(
		[]sql.Expression{
			expression.NewUnresolvedColumn("i"),
			expression.NewUnresolvedColumn("i2"),
		},
		plan.NewCrossJoin(
			plan.NewUnresolvedTable("mytable"),
			plan.NewUnresolvedTable("mytable2"),
		),
	)
	analyzed, err = a.Analyze(notAnalyzed)
	expected = plan.NewProject(
		[]sql.Expression{
			expression.NewGetFieldWithTable(0, sql.Int32, "mytable", "i", false),
			expression.NewGetFieldWithTable(1, sql.Int32, "mytable2", "i2", false),
		},
		plan.NewCrossJoin(table, table2),
	)
	require.NoError(err)
	require.Equal(expected, analyzed)

	notAnalyzed = plan.NewLimit(int64(1),
		plan.NewProject(
			[]sql.Expression{
				expression.NewUnresolvedColumn("i"),
			},
			plan.NewUnresolvedTable("mytable"),
		),
	)
	analyzed, err = a.Analyze(notAnalyzed)
	expected = plan.NewLimit(int64(1),
		plan.NewProject(
			[]sql.Expression{
				expression.NewGetFieldWithTable(0, sql.Int32, "mytable", "i", false),
			},
			table,
		),
	)
	require.Nil(err)
	require.Equal(expected, analyzed)
}

func TestAnalyzer_Analyze_MaxIterations(t *testing.T) {
	require := require.New(t)

	catalog := &sql.Catalog{}
	a := New(catalog)
	a.CurrentDatabase = "mydb"

	i := 0
	a.Rules = []Rule{{
		Name: "infinite",
		Apply: func(a *Analyzer, n sql.Node) (sql.Node, error) {
			i++
			return plan.NewUnresolvedTable(fmt.Sprintf("table%d", i)), nil
		},
	}}

	notAnalyzed := plan.NewUnresolvedTable("mytable")
	analyzed, err := a.Analyze(notAnalyzed)
	require.NotNil(err)
	require.Equal(plan.NewUnresolvedTable("table1001"), analyzed)
}
