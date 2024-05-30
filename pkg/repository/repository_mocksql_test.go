package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	qt "github.com/frankban/quicktest"
	"github.com/go-redis/redismock/v9"
	"github.com/gofrs/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func mockDBRepository() (sqlmock.Sqlmock, *sql.DB, Repository, error) {
	sqldb, mock, err := sqlmock.New()
	if err != nil {
		return nil, nil, nil, err
	}

	gormdb, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqldb,
	}))

	if err != nil {
		return nil, nil, nil, err
	}

	redisClient, _ := redismock.NewClientMock()
	repository := NewRepository(gormdb, redisClient)

	return mock, sqldb, repository, err
}

func TestRepository_CreatePipelineTags(t *testing.T) {
	c := qt.New(t)

	mock, sqldb, repository, err := mockDBRepository()
	defer sqldb.Close()
	c.Assert(err, qt.IsNil)
	uid, _ := uuid.NewV4()

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "tags" \("pipeline_uid","tag_name","create_time","update_time"\) VALUES \(\$1,\$2,\$3,\$4\),\(\$5,\$6,\$7,\$8\)`).
		WithArgs(uid, "tag1", sqlmock.AnyArg(), sqlmock.AnyArg(), uid, "tag2", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 2))
	mock.ExpectCommit()

	err = repository.CreatePipelineTags(context.Background(), uid, []string{"tag1", "tag2"})
	c.Assert(err, qt.IsNil)
}

func TestRepository_DeletePipelineTags(t *testing.T) {
	c := qt.New(t)

	mock, sqldb, repository, err := mockDBRepository()
	defer sqldb.Close()
	c.Assert(err, qt.IsNil)
	uid, _ := uuid.NewV4()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "tags" WHERE pipeline_uid = \$1 and tag_name in \(\$2,\$3\)`).
		WithArgs(uid, "tag1", "tag2").
		WillReturnResult(sqlmock.NewResult(1, 2))
	mock.ExpectCommit()

	err = repository.DeletePipelineTags(context.Background(), uid, []string{"tag1", "tag2"})
	c.Assert(err, qt.IsNil)
}

func TestRepository_ListPipelineTags(t *testing.T) {
	c := qt.New(t)

	mock, sqldb, repository, err := mockDBRepository()
	defer sqldb.Close()
	c.Assert(err, qt.IsNil)
	uid, _ := uuid.NewV4()

	now := time.Now()
	mock.ExpectQuery(`SELECT \* FROM "tags" WHERE pipeline_uid = \$1`).
		WithArgs(uid).
		WillReturnRows(
			sqlmock.NewRows([]string{"pipeline_uid", "tag_name", "create_time", "update_time"}).
				AddRow(uid, "tag1", now, now).
				AddRow(uid, "tag2", now, now))

	tags, err := repository.ListPipelineTags(context.Background(), uid)
	c.Assert(err, qt.IsNil)
	tagNames := make([]string, 0, len(tags))
	for _, tag := range tags {
		tagNames = append(tagNames, tag.TagName)
	}
	c.Assert(tagNames, qt.DeepEquals, []string{"tag1", "tag2"})
}
