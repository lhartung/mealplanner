package backend

import (
	"gopkg.in/guregu/null.v3"
	"gopkg.in/guregu/null.v3/zero"
)

/*
 * Changes:
 * 2 - Add Ingredient.Have
 * 3 - Add User.Admin
 * 4 - Add Family
 * 5 - Add FamilyMember
 * 8 - Add User.DefaultFamilyId
 * 10 - Add User.WeekStartDay
 * 12 - Add Family.CreatedOn
 * 14 - Add Family.AccountStatus
 * 16 - Add Family.StatusExpiresOn
 * 18 - Add Recipe.ImportId
 * 22 - Add User.Name
 * 23 - Add User.EmailVerified
 * 24 - Add User.EmailToken
 */

const ExpectDatabaseVersion int64 = 25

type Migration struct {
	Id      int64 `db:"id" json:"id"`
	Version int64 `db:"version" json:"version"`
	Time    int64 `db:"time" json:"time"`
}

type Family struct {
	Id     int64  `db:"id" json:"id"`
	UserId int64  `db:"user_id" json:"user_id"`
	Name   string `db:"name" json:"name"`

	CreatedOn       string `db:"created_on" json:"created_on"`
	AccountStatus   string `db:"account_status" json:"account_status"`
	StatusExpiresOn string `db:"status_expires_on" json:"status_expires_on"`
}

type FamilyMember struct {
	Id       int64 `db:"id" json:"id"`
	FamilyId int64 `db:"family_id" json:"family_id"`
	UserId   int64 `db:"user_id" json:"user_id"`
	CanEdit  bool  `db:"can_edit" json:"can_edit"`
}

type User struct {
	Id       int64  `db:"id" json:"id"`
	UserName string `db:"username" json:"username"` // Login name
	Password string `db:"password" json:"password"`
	Email    string `db:"email" json:"email"`
	Name     string `db:"name" json:"name"` // Display name

	Admin           zero.Bool `db:"admin" json:"admin"`
	DefaultFamilyId int64     `db:"default_family_id" json:"default_family_id"`
	WeekStartDay    int64     `db:"week_start_day" json:"week_start_day"`

	EmailVerified zero.Bool `db:"email_verified" json:"email_verified"`
	EmailToken    string    `db:"email_token" json:"email_token"`
}

type Recipe struct {
	Id       int64       `db:"id" json:"id"`
	OwnerId  int64       `db:"owner_id" json:"owner_id"`
	Show     bool        `db:"show" json:"show"`
	Fixed    bool        `db:"fixed" json:"fixed"`
	Name     string      `db:"name" json:"name"`
	Source   null.String `db:"source" json:"source"`
	Servings null.Int    `db:"servings" json:"servings"`
	ImportId int64       `db:"import_id" json:"import_id"`
}

type Ingredient struct {
	Id       int64       `db:"id" json:"id"`
	OwnerId  int64       `db:"owner_id" json:"owner_id"`
	RecipeId null.Int    `db:"recipe_id" json:"recipe_id"`
	ClassId  null.Int    `db:"class_id" json:"class_id"`
	Name     string      `db:"name" json:"name"`
	Amount   null.String `db:"amount" json:"amount"`
	Have     null.Bool   `db:"have" json:"have"`
}

type IngredientClass struct {
	Id       int64    `db:"id" json:"id"`
	ParentId null.Int `db:"parent_id" json:"parent_id"`
	Name     string   `db:"name" json:"name"`
}

type Assignment struct {
	Id       int64       `db:"id" json:"id"`
	OwnerId  int64       `db:"owner_id" json:"owner_id"`
	RecipeId null.Int    `db:"recipe_id" json:"recipe_id"`
	Date     null.String `db:"date" json:"date"`
	Meal     null.String `db:"meal" json:"meal"`
}
