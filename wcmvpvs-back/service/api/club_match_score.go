package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

var errClubMatchConflict = errors.New("match_changed")
var errClubMatchAction = errors.New("invalid_score_action")

type clubScoreAction struct {
	Action   string `json:"action"`
	Revision int64  `json:"revision"`
	BestOf   int    `json:"bestOf"`
}

func clubMatchWriteResult(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return errClubMatchConflict
	}
	return err
}

func (s *Store) applyClubScore(ctx context.Context, siteID, matchID int64, action clubScoreAction) (ClubMatch, error) {
	match, err := s.clubMatch(ctx, siteID, matchID)
	if err != nil {
		return match, err
	}
	if match.Revision != action.Revision {
		return match, errClubMatchConflict
	}
	if err := updateClubScore(&match, action); err != nil {
		return match, err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE club_matches
		SET status=?,score_for=?,score_against=?,sets_json=?,points_for=?,points_against=?,best_of=?,revision=revision+1
		WHERE site_id=? AND id=? AND revision=?`,
		match.Status, match.ScoreFor, match.ScoreAgainst, match.SetsJSON,
		match.PointsFor, match.PointsAgainst, match.BestOf, siteID, matchID, action.Revision)
	if err := clubMatchWriteResult(result, err); err != nil {
		return match, err
	}
	match.Revision++
	return match, nil
}

func updateClubScore(match *ClubMatch, action clubScoreAction) error {
	if action.Action == "start" {
		if (action.BestOf != 3 && action.BestOf != 5) ||
			(match.Status != "scheduled" && (match.Status != "live" || match.BestOf != 0)) {
			return errClubMatchAction
		}
		if match.Status == "scheduled" {
			match.ScoreFor, match.ScoreAgainst, match.SetsJSON = 0, 0, "[]"
		}
		match.Status, match.BestOf = "live", action.BestOf
		return nil
	}
	if match.Status != "live" || match.BestOf == 0 {
		return errClubMatchAction
	}

	setsWon := match.BestOf/2 + 1
	matchWon := match.ScoreFor >= setsWon || match.ScoreAgainst >= setsWon
	target := 25
	if match.ScoreFor+match.ScoreAgainst+1 == match.BestOf {
		target = 15
	}
	forPoints, againstPoints := match.PointsFor, match.PointsAgainst
	setWon := (forPoints >= target && forPoints-againstPoints >= 2) ||
		(againstPoints >= target && againstPoints-forPoints >= 2)
	switch action.Action {
	case "point_for", "point_against":
		if matchWon || setWon {
			return errClubMatchAction
		}
		if action.Action == "point_for" {
			match.PointsFor++
		} else {
			match.PointsAgainst++
		}
	case "undo_for":
		if forPoints == 0 {
			return errClubMatchAction
		}
		match.PointsFor--
	case "undo_against":
		if againstPoints == 0 {
			return errClubMatchAction
		}
		match.PointsAgainst--
	case "close_set", "reopen_set":
		if action.Action == "close_set" && (matchWon || !setWon) {
			return errClubMatchAction
		}
		return updateClubSet(match, action.Action == "reopen_set")
	case "finish":
		if !matchWon || forPoints != 0 || againstPoints != 0 {
			return errClubMatchAction
		}
		match.Status = "completed"
	default:
		return errClubMatchAction
	}
	return nil
}

func updateClubSet(match *ClubMatch, reopen bool) error {
	var sets []string
	if err := json.Unmarshal([]byte(match.SetsJSON), &sets); err != nil {
		return errClubMatchAction
	}
	if reopen {
		if len(sets) == 0 || match.PointsFor != 0 || match.PointsAgainst != 0 {
			return errClubMatchAction
		}
		last := strings.NewReplacer("–", "-", ":", "-").Replace(sets[len(sets)-1])
		left, right, _ := strings.Cut(last, "-")
		pointsFor, errFor := strconv.Atoi(strings.TrimSpace(left))
		pointsAgainst, errAgainst := strconv.Atoi(strings.TrimSpace(right))
		if errFor != nil || errAgainst != nil || pointsFor == pointsAgainst || pointsFor < 0 || pointsAgainst < 0 {
			return errClubMatchAction
		}
		if pointsFor > pointsAgainst && match.ScoreFor > 0 {
			match.ScoreFor--
		} else if pointsAgainst > pointsFor && match.ScoreAgainst > 0 {
			match.ScoreAgainst--
		} else {
			return errClubMatchAction
		}
		match.PointsFor, match.PointsAgainst = pointsFor, pointsAgainst
		sets = sets[:len(sets)-1]
	} else {
		sets = append(sets, fmt.Sprintf("%d-%d", match.PointsFor, match.PointsAgainst))
		if match.PointsFor > match.PointsAgainst {
			match.ScoreFor++
		} else {
			match.ScoreAgainst++
		}
		match.PointsFor, match.PointsAgainst = 0, 0
	}
	encoded, err := json.Marshal(sets)
	if err == nil {
		match.SetsJSON = string(encoded)
	}
	return err
}

func (rt *_router) clubAdminScore(w http.ResponseWriter, r *http.Request, siteID int64) {
	matchID, err := parseClubID(r, "id")
	var action clubScoreAction
	if err != nil || matchID <= 0 || json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&action) != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_score_action"})
		return
	}
	match, err := rt.store.applyClubScore(r.Context(), siteID, matchID, action)
	switch {
	case errors.Is(err, errClubNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "match_not_found"})
	case errors.Is(err, errClubMatchConflict):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "match_changed"})
	case errors.Is(err, errClubMatchAction):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "invalid_score_action"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "score_failed"})
	default:
		rt.clubMatches.publish(siteID, clubMatchMessage{Match: &match})
		writeJSON(w, http.StatusOK, match)
	}
}
