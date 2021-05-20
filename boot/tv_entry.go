package rkgin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/markbates/pkger"
	"github.com/rookie-ninja/rk-common/common"
	"github.com/rookie-ninja/rk-entry/entry"
	"github.com/rookie-ninja/rk-gin/interceptor/context"
	"github.com/rookie-ninja/rk-query"
	"go.uber.org/zap"
	"html/template"
	"io/ioutil"
	"net/http"
	"path"
	"strings"
	"time"
)

var (
	HeaderTemplate        = readFileFromPkger("/assets/tv/header.tmpl")
	FooterTemplate        = readFileFromPkger("/assets/tv/footer.tmpl")
	AsideTemplate         = readFileFromPkger("/assets/tv/aside.tmpl")
	HeadTemplate          = readFileFromPkger("/assets/tv/head.tmpl")
	SVGSpriteTemplate     = readFileFromPkger("/assets/tv/svg-sprite.tmpl")
	OverviewTemplate      = readFileFromPkger("/assets/tv/overview.tmpl")
	APITemplate           = readFileFromPkger("/assets/tv/api.tmpl")
	EntryTemplate         = readFileFromPkger("/assets/tv/entry.tmpl")
	ConfigTemplate        = readFileFromPkger("/assets/tv/config.tmpl")
	CertTemplate          = readFileFromPkger("/assets/tv/cert.tmpl")
	NotFoundTemplate      = readFileFromPkger("/assets/tv/not-found.tmpl")
	InternalErrorTemplate = readFileFromPkger("/assets/tv/internal-error.tmpl")
	OsTemplate            = readFileFromPkger("/assets/tv/os.tmpl")
	EnvTemplate           = readFileFromPkger("/assets/tv/env.tmpl")
	PrometheusTemplate    = readFileFromPkger("/assets/tv/prometheus.tmpl")
	LogTemplate           = readFileFromPkger("/assets/tv/log.tmpl")
)

const (
	TvEntryType        = "GinTvEntry"
	TvEntryNameDefault = "GinTvDefault"
	TvEntryDescription = "Internal RK entry which implements tv web with Gin framework."
)

// Read go template files with Pkger.
func readFileFromPkger(filePath string) []byte {
	if file, err := pkger.Open(path.Join("/boot", filePath)); err != nil {
		return []byte{}
	} else {
		if bytes, err := ioutil.ReadAll(file); err != nil {
			return []byte{}
		} else {
			return bytes
		}
	}
}

// Bootstrap config of tv.
// 1: Enabled: Enable tv service.
type BootConfigTv struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// RK TV entry supports web UI for application & process information.
// 1: EntryName: Name of entry.
// 2: EntryType: Type of entry.
// 2: EntryDescription: Description of entry.
// 3: ZapLoggerEntry: ZapLoggerEntry used for logging.
// 4: EventLoggerEntry: EventLoggerEntry used for logging.
// 5: Template: GO template for rendering web UI.
type TvEntry struct {
	EntryName        string                    `json:"entryName" yaml:"entryName"`
	EntryType        string                    `json:"entryType" yaml:"entryType"`
	EntryDescription string                    `json:"entryDescription" yaml:"entryDescription"`
	ZapLoggerEntry   *rkentry.ZapLoggerEntry   `json:"zapLoggerEntry" yaml:"zapLoggerEntry"`
	EventLoggerEntry *rkentry.EventLoggerEntry `json:"eventLoggerEntry" yaml:"eventLoggerEntry"`
	Template         *template.Template        `json:"-" yaml:"-"`
}

// TV entry option.
type TvEntryOption func(entry *TvEntry)

// Provide name.
func WithNameTv(name string) TvEntryOption {
	return func(entry *TvEntry) {
		entry.EntryName = name
	}
}

// Provide rkentry.EventLoggerEntry.
func WithEventLoggerEntryTv(eventLoggerEntry *rkentry.EventLoggerEntry) TvEntryOption {
	return func(entry *TvEntry) {
		entry.EventLoggerEntry = eventLoggerEntry
	}
}

// Provide rkentry.ZapLoggerEntry.
func WithZapLoggerEntryTv(zapLoggerEntry *rkentry.ZapLoggerEntry) TvEntryOption {
	return func(entry *TvEntry) {
		entry.ZapLoggerEntry = zapLoggerEntry
	}
}

// Create new TV entry with options.
func NewTvEntry(opts ...TvEntryOption) *TvEntry {
	entry := &TvEntry{
		EntryName:        TvEntryNameDefault,
		EntryType:        TvEntryType,
		EntryDescription: TvEntryDescription,
		ZapLoggerEntry:   rkentry.GlobalAppCtx.GetZapLoggerEntryDefault(),
		EventLoggerEntry: rkentry.GlobalAppCtx.GetEventLoggerEntryDefault(),
	}

	for i := range opts {
		opts[i](entry)
	}

	if len(entry.EntryName) < 1 {
		entry.EntryName = TvEntryNameDefault
	}

	return entry
}

// Handler which returns js, css, images and html files for TV web UI.
func (entry *TvEntry) AssetsFileHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		w := ctx.Writer
		r := ctx.Request

		p := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/rk/v1"), "/")

		if file, err := pkger.Open(path.Join("/boot", p)); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		} else {
			http.ServeContent(w, r, path.Base(p), time.Now(), file)
		}
	}
}

// Bootstrap TV entry.
// Rendering bellow templates.
// 1: head.tmpl
// 2: header.tmpl
// 3: footer.tmpl
// 4: aside.tmpl
// 5: svg-sprite.tmpl
// 6: overview.tmpl
// 7: api.tmpl
// 8: entry.tmpl
// 9: config.tmpl
// 10: cert.tmpl
// 11: os.tmpl
// 12: env.tmpl
// 13: prometheus.tmpl
// 14: log.tmpl
func (entry *TvEntry) Bootstrap(ctx context.Context) {
	event := entry.EventLoggerEntry.GetEventHelper().Start(
		"bootstrap",
		rkquery.WithEntryName(entry.EntryName),
		rkquery.WithEntryType(entry.EntryType))

	entry.logBasicInfo(event)

	entry.ZapLoggerEntry.GetLogger().Info("Bootstrapping TvEntry.", event.GetFields()...)

	event.AddFields(zap.String("path", "/rk/v1/tv/*item"))

	entry.Template = template.New("rk-tv")

	// Parse head template
	if _, err := entry.Template.Parse(string(HeadTemplate)); err != nil {
		entry.EventLoggerEntry.GetEventHelper().FinishWithError(event, err)
		entry.ZapLoggerEntry.GetLogger().Error("Error occurs while parsing head template.")
		rkcommon.ShutdownWithError(err)
	}

	// Parse header template
	if _, err := entry.Template.Parse(string(HeaderTemplate)); err != nil {
		entry.EventLoggerEntry.GetEventHelper().FinishWithError(event, err)
		entry.ZapLoggerEntry.GetLogger().Error("Error occurs while parsing header template.")
		rkcommon.ShutdownWithError(err)
	}

	// Parse footer template
	if _, err := entry.Template.Parse(string(FooterTemplate)); err != nil {
		entry.EventLoggerEntry.GetEventHelper().FinishWithError(event, err)
		entry.ZapLoggerEntry.GetLogger().Error("Error occurs while parsing footer template.")
		rkcommon.ShutdownWithError(err)
	}

	// Parse aside template
	if _, err := entry.Template.Parse(string(AsideTemplate)); err != nil {
		entry.EventLoggerEntry.GetEventHelper().FinishWithError(event, err)
		entry.ZapLoggerEntry.GetLogger().Error("Error occurs while parsing aside template.")
		rkcommon.ShutdownWithError(err)
	}

	// Parse svg-sprite template
	if _, err := entry.Template.Parse(string(SVGSpriteTemplate)); err != nil {
		entry.EventLoggerEntry.GetEventHelper().FinishWithError(event, err)
		entry.ZapLoggerEntry.GetLogger().Error("Error occurs while parsing svg-sprite template.")
		rkcommon.ShutdownWithError(err)
	}

	// Parse overview template
	if _, err := entry.Template.Parse(string(OverviewTemplate)); err != nil {
		fmt.Println(string(OverviewTemplate))

		entry.EventLoggerEntry.GetEventHelper().FinishWithError(event, err)
		entry.ZapLoggerEntry.GetLogger().Error("Error occurs while overview template.")
		rkcommon.ShutdownWithError(err)
	}

	// Parse api template
	if _, err := entry.Template.Parse(string(APITemplate)); err != nil {
		entry.EventLoggerEntry.GetEventHelper().FinishWithError(event, err)
		entry.ZapLoggerEntry.GetLogger().Error("Error occurs while api template.")
		rkcommon.ShutdownWithError(err)
	}

	// Parse entry template
	if _, err := entry.Template.Parse(string(EntryTemplate)); err != nil {
		entry.EventLoggerEntry.GetEventHelper().FinishWithError(event, err)
		entry.ZapLoggerEntry.GetLogger().Error("Error occurs while entry template.")
		rkcommon.ShutdownWithError(err)
	}

	// Parse config template
	if _, err := entry.Template.Parse(string(ConfigTemplate)); err != nil {
		entry.EventLoggerEntry.GetEventHelper().FinishWithError(event, err)
		entry.ZapLoggerEntry.GetLogger().Error("Error occurs while config template.")
		rkcommon.ShutdownWithError(err)
	}

	// Parse cert template
	if _, err := entry.Template.Parse(string(CertTemplate)); err != nil {
		entry.EventLoggerEntry.GetEventHelper().FinishWithError(event, err)
		entry.ZapLoggerEntry.GetLogger().Error("Error occurs while cert template.")
		rkcommon.ShutdownWithError(err)
	}

	// Parse os template
	if _, err := entry.Template.Parse(string(OsTemplate)); err != nil {
		entry.EventLoggerEntry.GetEventHelper().FinishWithError(event, err)
		entry.ZapLoggerEntry.GetLogger().Error("Error occurs while os template.")
		rkcommon.ShutdownWithError(err)
	}

	// Parse env template
	if _, err := entry.Template.Parse(string(EnvTemplate)); err != nil {
		entry.EventLoggerEntry.GetEventHelper().FinishWithError(event, err)
		entry.ZapLoggerEntry.GetLogger().Error("Error occurs while env template.")
		rkcommon.ShutdownWithError(err)
	}

	// Parse prometheus template
	if _, err := entry.Template.Parse(string(PrometheusTemplate)); err != nil {
		entry.EventLoggerEntry.GetEventHelper().FinishWithError(event, err)
		entry.ZapLoggerEntry.GetLogger().Error("Error occurs while prometheus template.")
		rkcommon.ShutdownWithError(err)
	}

	// Parse log template
	if _, err := entry.Template.Parse(string(LogTemplate)); err != nil {
		entry.EventLoggerEntry.GetEventHelper().FinishWithError(event, err)
		entry.ZapLoggerEntry.GetLogger().Error("Error occurs while log template.")
		rkcommon.ShutdownWithError(err)
	}

	// Parse not found template
	if _, err := entry.Template.Parse(string(NotFoundTemplate)); err != nil {
		entry.EventLoggerEntry.GetEventHelper().FinishWithError(event, err)
		entry.ZapLoggerEntry.GetLogger().Error("Error occurs while not-found template.")
		rkcommon.ShutdownWithError(err)
	}

	// Parse internal server template
	if _, err := entry.Template.Parse(string(InternalErrorTemplate)); err != nil {
		entry.EventLoggerEntry.GetEventHelper().FinishWithError(event, err)
		entry.ZapLoggerEntry.GetLogger().Error("Error occurs while internal-server template.")
		rkcommon.ShutdownWithError(err)
	}

	entry.ZapLoggerEntry.GetLogger().Info("Bootstrapping tvEntry.", event.GetFields()...)

	entry.EventLoggerEntry.GetEventHelper().Finish(event)
}

// Interrupt TV entry.
func (entry *TvEntry) Interrupt(context.Context) {
	event := entry.EventLoggerEntry.GetEventHelper().Start(
		"interrupt",
		rkquery.WithEntryName(entry.EntryName),
		rkquery.WithEntryType(entry.EntryType))

	entry.logBasicInfo(event)

	defer entry.EventLoggerEntry.GetEventHelper().Finish(event)

	entry.ZapLoggerEntry.GetLogger().Info("Interrupting TvEntry.", event.GetFields()...)
}

// Get name of entry.
func (entry *TvEntry) GetName() string {
	return entry.EntryName
}

// Get type of entry.
func (entry *TvEntry) GetType() string {
	return entry.EntryType
}

// Get description of entry.
func (entry *TvEntry) GetDescription() string {
	return entry.EntryDescription
}

// Stringfy entry.
func (entry *TvEntry) String() string {
	bytesStr, _ := json.Marshal(entry)
	return string(bytesStr)
}

// Marshal entry
func (entry *TvEntry) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"entryName":        entry.EntryName,
		"entryType":        entry.EntryType,
		"entryDescription": entry.EntryDescription,
		"eventLoggerEntry": entry.EventLoggerEntry.GetName(),
		"zapLoggerEntry":   entry.ZapLoggerEntry.GetName(),
	}

	return json.Marshal(&m)
}

// Not supported.
func (entry *TvEntry) UnmarshalJSON([]byte) error {
	return nil
}

// Add basic fields into event.
func (entry *TvEntry) logBasicInfo(event rkquery.Event) {
	event.AddFields(
		zap.String("entryName", entry.EntryName),
		zap.String("entryType", entry.EntryType),
	)
}

// @Summary Get HTML page of /tv
// @Id 11
// @version 1.0
// @produce text/html
// @Success 200 string HTML
// @Router /rk/v1/tv [get]
func (entry *TvEntry) TV(ctx *gin.Context) {
	if ctx == nil {
		return
	}

	// Add auto generated request ID
	rkginctx.AddRequestIdToOutgoingHeader(ctx)

	logger := rkginctx.GetLogger(ctx)

	contentType := "text/html; charset=utf-8"

	switch item := ctx.Param("item"); item {
	case "/", "/overview":
		buf := new(bytes.Buffer)

		if err := entry.Template.ExecuteTemplate(buf, "overview", doInfo(ctx)); err != nil {
			logger.Warn("Failed to execute template", zap.Error(err))
			buf.Reset()
			entry.Template.ExecuteTemplate(buf, "internal-error", nil)
		}
		ctx.Data(http.StatusOK, contentType, buf.Bytes())
	case "/api":
		buf := new(bytes.Buffer)

		if err := entry.Template.ExecuteTemplate(buf, "api", doApis(ctx)); err != nil {
			logger.Warn("Failed to execute template", zap.Error(err))
			buf.Reset()
			entry.Template.ExecuteTemplate(buf, "internal-error", nil)
		}

		ctx.Data(http.StatusOK, contentType, buf.Bytes())
	case "/entry":
		buf := new(bytes.Buffer)

		if err := entry.Template.ExecuteTemplate(buf, "entry", doEntries(ctx)); err != nil {
			logger.Warn("Failed to execute template", zap.Error(err))
			buf.Reset()
			entry.Template.ExecuteTemplate(buf, "internal-error", nil)
		}
		ctx.Data(http.StatusOK, contentType, buf.Bytes())
	case "/config":
		buf := new(bytes.Buffer)

		if err := entry.Template.ExecuteTemplate(buf, "config", doConfigs(ctx)); err != nil {
			logger.Warn("Failed to execute template", zap.Error(err))
			buf.Reset()
			entry.Template.ExecuteTemplate(buf, "internal-error", nil)
		}
		ctx.Data(http.StatusOK, contentType, buf.Bytes())
	case "/cert":
		buf := new(bytes.Buffer)

		if err := entry.Template.ExecuteTemplate(buf, "cert", doCerts(ctx)); err != nil {
			logger.Warn("Failed to execute template", zap.Error(err))
			buf.Reset()
			entry.Template.ExecuteTemplate(buf, "internal-error", nil)
		}
		ctx.Data(http.StatusOK, contentType, buf.Bytes())
	case "/os":
		buf := new(bytes.Buffer)

		if err := entry.Template.ExecuteTemplate(buf, "os", doSys(ctx)); err != nil {
			logger.Warn("Failed to execute template", zap.Error(err))
			buf.Reset()
			entry.Template.ExecuteTemplate(buf, "internal-error", nil)
		}
		ctx.Data(http.StatusOK, contentType, buf.Bytes())
	case "/env":
		buf := new(bytes.Buffer)

		if err := entry.Template.ExecuteTemplate(buf, "env", doSys(ctx)); err != nil {
			logger.Warn("Failed to execute template", zap.Error(err))
			buf.Reset()
			entry.Template.ExecuteTemplate(buf, "internal-error", nil)
		}
		ctx.Data(http.StatusOK, contentType, buf.Bytes())
	case "/prometheus":
		buf := new(bytes.Buffer)

		if err := entry.Template.ExecuteTemplate(buf, "prometheus", nil); err != nil {
			logger.Warn("Failed to execute template", zap.Error(err))
			buf.Reset()
			entry.Template.ExecuteTemplate(buf, "internal-error", nil)
		}
		ctx.Data(http.StatusOK, contentType, buf.Bytes())
	case "/log":
		buf := new(bytes.Buffer)

		if err := entry.Template.ExecuteTemplate(buf, "log", doLogs(ctx)); err != nil {
			logger.Warn("Failed to execute template", zap.Error(err))
			buf.Reset()
			entry.Template.ExecuteTemplate(buf, "internal-error", nil)
		}
		ctx.Data(http.StatusOK, contentType, buf.Bytes())
	default:
		buf := new(bytes.Buffer)

		if err := entry.Template.ExecuteTemplate(buf, "not-found", nil); err != nil {
			logger.Warn("Failed to execute template", zap.Error(err))
			buf.Reset()
			entry.Template.ExecuteTemplate(buf, "internal-error", nil)
		}
		ctx.Data(http.StatusOK, contentType, buf.Bytes())
	}
}
