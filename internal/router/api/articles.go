package api

import (
	"fmt"
	"net/http"

	"github.com/ChiaYuChang/weathercock/internal/models"
	"github.com/ChiaYuChang/weathercock/pkgs/errors"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type PublicArticles struct {
	*Repo
	*validator.Validate
}

func (r *Repo) PublicArticles(validator *validator.Validate) *PublicArticles {
	return &PublicArticles{
		Repo:     r,
		Validate: validator,
	}
}

type UserArticles struct {
	*Repo
	*validator.Validate
}

func (r *Repo) UserArticles(validator *validator.Validate) *UserArticles {
	return &UserArticles{
		Repo:     r,
		Validate: validator,
	}
}

func (a *UserArticles) GetByTaskID(r *http.Request) (*models.UsersArticle, error) {
	taskID, err := uuid.Parse(r.PathValue("task_id"))
	if err != nil {
		e := errors.ErrBadRequest.Clone().
			WithDetails("invalid task_id format").
			Warp(err)
		return nil, e
	}

	if err := a.Validate.VarCtx(r.Context(), taskID, "uuid4,required"); err != nil {
		e := errors.ErrBadRequest.Clone().
			WithDetails("task id should be a valid UUID (ver 4)").
			Warp(err)
		return nil, e
	}

	article, err := a.Storage.Queries.GetUsersArticleByTaskID(r.Context(), taskID)
	if err != nil {
		pge, ok := errors.NewPGErr(err)
		var e *errors.Error
		if ok {
			e = errors.FromPgError(pge)
		} else {
			e = errors.ErrDBError.Clone().
				Warp(err).
				WithDetails(fmt.Sprintf("failed to get user article by task_id: %s", taskID.String()))
		}
		return nil, e
	}
	return &article, nil
}
