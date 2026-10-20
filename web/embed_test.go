package web

import (
	"io/fs"
	"strings"
	"testing"
)

func TestSourceLinksOnlyAppearAfterQuizAndExamEvaluation(t *testing.T) {
	data, err := fs.ReadFile(Files, "app.js")
	if err != nil {
		t.Fatal(err)
	}
	script := string(data)
	examQuestion := functionBody(t, script, "function renderExamQuestion()", "function bindExamEvents")
	if strings.Contains(examQuestion, "topicSourceAction") || strings.Contains(examQuestion, "Originalfolie") {
		t.Fatal("running exam renders a source link")
	}
	examResult := functionBody(t, script, "function renderExamResult()", "function practiceExamView")
	if !strings.Contains(examResult, "topicSourceAction") {
		t.Fatal("finished exam does not render source links")
	}
	quiz := functionBody(t, script, "function renderQuestion()", "function bindQuestionEvents")
	locked := strings.Index(quiz, "${locked ?")
	source := strings.Index(quiz, "topicSourceAction")
	if locked < 0 || source < locked {
		t.Fatal("quiz source link is not restricted to evaluated feedback")
	}
}

func TestLearningPageRendersVisibleSeparateLinksForPageRanges(t *testing.T) {
	data, err := fs.ReadFile(Files, "app.js")
	if err != nil {
		t.Fatal(err)
	}
	topic := functionBody(t, string(data), "function topicView(topicID)", "function todaySetupView")
	for _, marker := range []string{"original-material-heading", "source-list", "source.page_ranges", "pageRanges.map", "range.label", "range.url", "source-range", "Originalfolien öffnen"} {
		if !strings.Contains(topic, marker) {
			t.Fatalf("learning page does not render page ranges using %q", marker)
		}
	}
	if strings.Contains(topic, "<details>") {
		t.Fatal("learning page hides original material in a collapsed disclosure")
	}
}

func TestTopicNavigationIsAtPageEndAndUsesNativeLinks(t *testing.T) {
	scriptBytes, err := fs.ReadFile(Files, "app.js")
	if err != nil {
		t.Fatal(err)
	}
	stylesBytes, err := fs.ReadFile(Files, "styles.css")
	if err != nil {
		t.Fatal(err)
	}
	topic := functionBody(t, string(scriptBytes), "function topicView(topicID)", "function todaySetupView")
	for _, marker := range []string{"topicNavigation(catalog.topics, topicID)", "sequence.previousID", "sequence.nextID", "Thema ${sequence.position} / ${sequence.total}", "data-topic-previous", "data-topic-training", "data-topic-next", "Zum Prüfungsbereich", "#/quiz/${exam.id}/${module.id}", "#/exam/${exam.id}"} {
		if !strings.Contains(topic, marker) {
			t.Errorf("topic navigation missing %q", marker)
		}
	}
	if !strings.Contains(topic, `${navigation}</article>`) {
		t.Fatal("navigation must finish the topic article, below sources")
	}
	if !strings.Contains(topic, `<nav class="topic-navigation" aria-label="Themennavigation">`) {
		t.Fatal("native navigation landmark missing")
	}
	styles := functionBody(t, string(stylesBytes), ".topic-navigation {", ".core-idea {")
	if strings.Contains(styles, "position: fixed") || strings.Contains(styles, "position: sticky") {
		t.Fatal("topic controls must not overlay content")
	}
	for _, marker := range []string{"min-height: 48px", "grid-template-columns: 1fr"} {
		if !strings.Contains(styles, marker) {
			t.Errorf("missing mobile navigation style %q", marker)
		}
	}
}

func TestPDFsAreNotEmbedded(t *testing.T) {
	matches, err := fs.Glob(Files, "*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("embedded PDFs: %v", matches)
	}
}

func TestFaviconAssetsAreEmbeddedAndLinked(t *testing.T) {
	indexBytes, err := fs.ReadFile(Files, "index.html")
	if err != nil {
		t.Fatal(err)
	}
	index := string(indexBytes)
	for _, marker := range []string{"/favicon.svg", "/favicon.ico", "/apple-touch-icon.png"} {
		if !strings.Contains(index, marker) {
			t.Fatalf("index does not link favicon asset %q", marker)
		}
	}
	for _, file := range []string{"favicon.svg", "favicon.ico", "apple-touch-icon.png"} {
		data, err := fs.ReadFile(Files, file)
		if err != nil || len(data) == 0 {
			t.Fatalf("embedded favicon asset %q unavailable: %v", file, err)
		}
	}
}

func TestHeaderBrandUsesTheNodeMark(t *testing.T) {
	indexBytes, err := fs.ReadFile(Files, "index.html")
	if err != nil {
		t.Fatal(err)
	}
	index := string(indexBytes)
	for _, marker := range []string{
		"<svg class=\"brand-mark\"",
		"fill=\"#0B1220\"",
		"stroke=\"#38BDF8\"",
		"cx=\"19\" cy=\"22\"",
		"cx=\"45\" cy=\"22\"",
		"cx=\"32\" cy=\"44\"",
	} {
		if !strings.Contains(index, marker) {
			t.Fatalf("header brand is missing node-mark detail %q", marker)
		}
	}
	if strings.Contains(index, "class=\"brand-mark\" aria-hidden=\"true\">ML") {
		t.Fatal("header brand still uses the former ML text mark")
	}
}

func TestPhaseSevenAccessibilityStructureAndStatusMessages(t *testing.T) {
	indexBytes, err := fs.ReadFile(Files, "index.html")
	if err != nil {
		t.Fatal(err)
	}
	scriptBytes, err := fs.ReadFile(Files, "app.js")
	if err != nil {
		t.Fatal(err)
	}
	index, script := string(indexBytes), string(scriptBytes)
	for _, marker := range []string{"<nav", "<main", "<fieldset", "<legend", "class=\"skip-link\"", "id=\"global-status\"", "role=\"status\"", "aria-live=\"polite\""} {
		if !strings.Contains(index, marker) {
			t.Fatalf("frontend shell missing accessibility marker %q", marker)
		}
	}
	for _, marker := range []string{"<dialog", "aria-labelledby=\"restore-title\"", "role=\"alert\"", "dialog.showModal()", "dialog.addEventListener('cancel'", "dialog.addEventListener('close'", "focusHeading()", "announce("} {
		if !strings.Contains(script, marker) {
			t.Fatalf("frontend behavior missing accessibility marker %q", marker)
		}
	}
	if strings.Contains(script, "role=\"button\"") || strings.Contains(script, "draggable=") {
		t.Fatal("frontend uses a simulated button or drag-only control")
	}
}

func TestAllQuizControlsHaveNativeKeyboardPaths(t *testing.T) {
	scriptBytes, err := fs.ReadFile(Files, "app.js")
	if err != nil {
		t.Fatal(err)
	}
	script := string(scriptBytes)
	control := functionBody(t, script, "function questionControl", "function focusOrderControl")
	for _, marker := range []string{"multiple ? 'checkbox' : 'radio'", "<select", "<button type=\"button\" data-move=\"up\"", "<button type=\"button\" data-move=\"down\"", "aria-label="} {
		if !strings.Contains(control, marker) {
			t.Fatalf("quiz controls missing native keyboard path %q", marker)
		}
	}
	if !strings.Contains(script, "focusOrderControl(target, direction)") {
		t.Fatal("ordering rerender does not restore keyboard focus")
	}
}

func TestExamScopeCopyReflectsCoreAndSupportContent(t *testing.T) {
	data, err := fs.ReadFile(Files, "app.js")
	if err != nil {
		t.Fatal(err)
	}
	script := string(data)
	if strings.Contains(script, "Die acht derzeit befüllten Pilot-Themen") || strings.Contains(script, "momentan identisch") {
		t.Fatal("exam configuration still claims all and core scopes are identical")
	}
	for _, marker := range []string{"Alle Themen berücksichtigt Kernstoff und Ergänzungen", "Nur Kernstoff beschränkt die Auswahl auf Kernthemen"} {
		if !strings.Contains(script, marker) {
			t.Fatalf("exam scope copy missing %q", marker)
		}
	}
}

func TestModuleOverviewUsesLoadedTopicsForContentFallback(t *testing.T) {
	data, err := fs.ReadFile(Files, "app.js")
	if err != nil {
		t.Fatal(err)
	}
	modules := functionBody(t, string(data), "function examView(examID)", "function moduleView(moduleID)")
	for _, marker := range []string{"const topics = topicsFor(module.id)", "topics.length ?", "'Content folgt'", "href=\"#/module/${module.id}\""} {
		if !strings.Contains(modules, marker) {
			t.Fatalf("module overview is not driven by loaded topic count: missing %q", marker)
		}
	}
	for _, forbidden := range []string{"hasContent", "enabledModules", "sampleTopics", "comingSoon"} {
		if strings.Contains(modules, forbidden) {
			t.Fatalf("module overview contains stale pilot flag %q", forbidden)
		}
	}
}

func TestTopicViewDefensivelyNormalizesOptionalCollections(t *testing.T) {
	data, err := fs.ReadFile(Files, "app.js")
	if err != nil {
		t.Fatal(err)
	}
	topic := functionBody(t, string(data), "function topicView(topicID)", "function todaySetupView")
	for _, marker := range []string{
		"Array.isArray(topic.steps) ? topic.steps : []",
		"Array.isArray(topic.confusions) ? topic.confusions : []",
		"Array.isArray(topic.sources) ? topic.sources : []",
		"renderFormula(topic.formula)",
		"steps.length",
		"confusions.map",
	} {
		if !strings.Contains(topic, marker) {
			t.Fatalf("topic view is not defensive for optional collection: missing %q", marker)
		}
	}
}

func TestResponsiveFocusMotionAndTouchStyles(t *testing.T) {
	styleBytes, err := fs.ReadFile(Files, "styles.css")
	if err != nil {
		t.Fatal(err)
	}
	styles := string(styleBytes)
	for _, marker := range []string{":focus-visible", "outline: 3px solid var(--primary)", "@media (max-width: 820px)", "@media (max-width: 430px)", "@media (prefers-reduced-motion: reduce)", "grid-template-columns: repeat(5, 1fr)", "min-height: 44px", ".restore-dialog::backdrop"} {
		if !strings.Contains(styles, marker) {
			t.Fatalf("styles missing phase-seven marker %q", marker)
		}
	}
	if strings.Contains(styles, "outline: none") || strings.Contains(styles, "outline: 0") {
		t.Fatal("styles remove keyboard focus outlines")
	}
}

func TestBackupRestoreUIWarnsAndUsesBoundedJSONFlow(t *testing.T) {
	scriptBytes, err := fs.ReadFile(Files, "app.js")
	if err != nil {
		t.Fatal(err)
	}
	script := string(scriptBytes)
	settings := functionBody(t, script, "function settingsView()", "function notFoundView")
	for _, marker := range []string{"/api/backup", "/api/restore", "Dieses Backup ersetzt deinen aktuellen Lernfortschritt", "8 * 1024 * 1024", "Content-Type': 'application/json'", "✓ Backup erfolgreich wiederhergestellt", "Sicherheitskopie"} {
		if !strings.Contains(settings, marker) {
			t.Fatalf("backup/restore UI missing %q", marker)
		}
	}
}

func TestFrontendHasNoInlineStylesOrExternalRuntimeDependencies(t *testing.T) {
	for _, name := range []string{"index.html", "ui-state.js", "learning-visuals.js", "formula-renderer.js", "app.js", "styles.css"} {
		data, err := fs.ReadFile(Files, name)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if strings.Contains(text, "style=") {
			t.Fatalf("%s contains an inline style incompatible with strict CSP", name)
		}
		withoutSVGNamespace := strings.ReplaceAll(text, "http://www.w3.org/2000/svg", "")
		if strings.Contains(withoutSVGNamespace, "https://") || strings.Contains(withoutSVGNamespace, "http://") {
			t.Fatalf("%s contains an external runtime URL", name)
		}
	}
}

func TestInitialAuthStateKeepsProtectedShellHiddenUntilServerResult(t *testing.T) {
	indexBytes, err := fs.ReadFile(Files, "index.html")
	if err != nil {
		t.Fatal(err)
	}
	scriptBytes, err := fs.ReadFile(Files, "app.js")
	if err != nil {
		t.Fatal(err)
	}
	styleBytes, err := fs.ReadFile(Files, "styles.css")
	if err != nil {
		t.Fatal(err)
	}
	index, script, styles := string(indexBytes), string(scriptBytes), string(styleBytes)
	for _, marker := range []string{`<body data-auth-state="unknown">`, `class="auth-loading"`, `Sitzung wird geprüft`, `/ui-state.js?v=17.1`} {
		if !strings.Contains(index, marker) {
			t.Fatalf("initial HTML missing %q", marker)
		}
	}
	for _, marker := range []string{`body[data-auth-state="unknown"] .sidebar`, `body[data-auth-state="unknown"] .topbar`, `body[data-auth-state="unknown"] .bottom-nav`, `#app > :not(.auth-loading)`} {
		if !strings.Contains(styles, marker) {
			t.Fatalf("unknown auth state is not visually isolated: missing %q", marker)
		}
	}
	for _, marker := range []string{`let authState = AUTH.UNKNOWN`, `initializeApplication()`, `await requestJSON('/api/auth/status')`, `if (!status.authenticated) { renderLogin(); return; }`, `await startApplication('session-valid')`} {
		if !strings.Contains(script, marker) {
			t.Fatalf("auth initialization missing %q", marker)
		}
	}
	if strings.Count(script, "/api/auth/status") != 1 {
		t.Fatal("frontend performs parallel or duplicate initial auth checks")
	}
	if strings.Contains(script, "setTimeout(") {
		t.Fatal("auth initialization uses a timing workaround")
	}
}

func TestMobileLogoutAndDesktopLogoutShareOneImplementation(t *testing.T) {
	scriptBytes, err := fs.ReadFile(Files, "app.js")
	if err != nil {
		t.Fatal(err)
	}
	stylesBytes, err := fs.ReadFile(Files, "styles.css")
	if err != nil {
		t.Fatal(err)
	}
	script, styles := string(scriptBytes), string(stylesBytes)
	for _, marker := range []string{`id="mobile-logout-button"`, `>Abmelden</button>`, `addEventListener('click', logout)`, `async function logout(event)`, `authTransition(authState, 'logout-succeeded')`} {
		if !strings.Contains(script, marker) {
			t.Fatalf("shared logout flow missing %q", marker)
		}
	}
	for _, marker := range []string{`.session-settings { display: none; }`, `.session-settings { display: block; }`, `.mobile-logout { width: 100%;`, `min-height: 44px`} {
		if !strings.Contains(styles, marker) {
			t.Fatalf("mobile logout styling missing %q", marker)
		}
	}
}

func TestThemeControlsExposeAndPersistSystemLightDark(t *testing.T) {
	indexBytes, err := fs.ReadFile(Files, "index.html")
	if err != nil {
		t.Fatal(err)
	}
	scriptBytes, err := fs.ReadFile(Files, "app.js")
	if err != nil {
		t.Fatal(err)
	}
	stylesBytes, err := fs.ReadFile(Files, "styles.css")
	if err != nil {
		t.Fatal(err)
	}
	index, script, styles := string(indexBytes), string(scriptBytes), string(stylesBytes)
	for _, value := range []string{"system", "light", "dark"} {
		if !strings.Contains(index, `data-theme-choice="`+value+`"`) || !strings.Contains(script, `value="${value}" data-theme-choice="${value}"`) {
			t.Fatalf("theme %q is not available on desktop and settings", value)
		}
	}
	for _, marker := range []string{`type="radio" name="settings-theme"`, `await requestJSON('/api/settings/theme'`, `root.dataset.themePreference`, `systemTheme.addEventListener('change'`, `if (root.dataset.themePreference === 'system')`} {
		if !strings.Contains(script, marker) {
			t.Fatalf("theme behavior missing %q", marker)
		}
	}
	if !strings.Contains(styles, `@media (prefers-color-scheme: dark)`) || !strings.Contains(styles, `:root[data-theme="system"]`) {
		t.Fatal("system theme has no early CSS fallback")
	}
}

func TestVersionThirteenFooterUsesCentralVersionSource(t *testing.T) {
	indexBytes, err := fs.ReadFile(Files, "index.html")
	if err != nil {
		t.Fatal(err)
	}
	scriptBytes, err := fs.ReadFile(Files, "app.js")
	if err != nil {
		t.Fatal(err)
	}
	stateBytes, err := fs.ReadFile(Files, "ui-state.js")
	if err != nil {
		t.Fatal(err)
	}
	stylesBytes, err := fs.ReadFile(Files, "styles.css")
	if err != nil {
		t.Fatal(err)
	}
	index, script, state, styles := string(indexBytes), string(scriptBytes), string(stateBytes), string(stylesBytes)
	for _, marker := range []string{"?v=17.1", `id="app-footer"`, `id="app-version"`, `aria-label="Anwendungsversion"`} {
		if !strings.Contains(index, marker) {
			t.Fatalf("v1.7.1 footer metadata is missing %q", marker)
		}
	}
	for _, marker := range []string{`const APP_VERSION = '1.7.1'`, `Object.freeze({ APP_VERSION,`, `appVersion.textContent = ` + "`ML/DL Exam Trainer · v${APP_VERSION}`"} {
		if !strings.Contains(state+script, marker) {
			t.Fatalf("central version source is missing %q", marker)
		}
	}
	if strings.Contains(script, "1.7.1") {
		t.Fatal("application version is duplicated in app.js")
	}
	footerRule := functionBody(t, styles, ".app-footer {", "body[data-auth-state=\"unknown\"]")
	if !strings.Contains(footerRule, "color: var(--text-secondary)") || !strings.Contains(footerRule, "font-size: 13px") {
		t.Fatal("footer is not styled as a small readable secondary element")
	}
	if strings.Contains(footerRule, "position: fixed") || strings.Contains(footerRule, "position: sticky") {
		t.Fatal("footer must remain in normal document flow")
	}
}

func functionBody(t *testing.T, script, startMarker, endMarker string) string {
	t.Helper()
	start := strings.Index(script, startMarker)
	end := strings.Index(script, endMarker)
	if start < 0 || end <= start {
		t.Fatalf("cannot locate %q to %q", startMarker, endMarker)
	}
	return script[start:end]
}
