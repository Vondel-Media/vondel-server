package handlers

import (
	"net/http"

	"github.com/Vondel-Media/vondel-server/internal/activitylog"
)

func setPlaybackSessionLogContext(r *http.Request, sessionID string) {
	if sessionID == "" {
		return
	}
	if lc := activitylog.GetPlaybackLogContext(r.Context()); lc != nil {
		lc.PlaybackSessionID = sessionID
	}
}
