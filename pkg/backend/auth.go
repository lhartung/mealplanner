package backend

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/sessions"
)

type AuthResponse struct {
	Message  string        `json:"message"`
	Token    string        `json:"token"`
	UserId   int64         `json:"user_id"`
	Families []int64       `json:"families"`
}

type FamilyAccess struct {
	Family
}

func logError(err error) {
	fmt.Printf("Error: %v\n", err)
}

func checkFamilyParam(params martini.Params, session sessions.Session) (int64, bool) {
	family_id, err := strconv.ParseInt(params["family_id"], 0, 64)
	if err != nil {
		logError(err)
		return family_id, false
	}

	admin := session.Get("Admin")
	if admin != nil && admin.(bool) {
		return family_id, true
	}

	families, ok := session.Get("Families").([]int64)
	if !ok {
		return family_id, false
	}

	index := sort.Search(len(families), func(i int) bool { return families[i] == family_id })
	return family_id, (index < len(families) && families[index] == family_id)
}
