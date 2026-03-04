package service

import (
	"5000blogs/config"
	"archive/zip"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
)

func RunInitialSetup(cfg *config.Config, logger *slog.Logger) error {
	log := logger.With("component", "setup")

	if err := ensureTemplate(cfg, log); err != nil {
		return err
	}

	return nil
}

// ensureTemplate downloads and extracts the template zip if needed
func ensureTemplate(cfg *config.Config, log *slog.Logger) error {
	templatePath := filepath.Join(cfg.Paths.Static, "template.html")

	if _, err := os.Stat(templatePath); err == nil {
		// Already present – nothing to do.
		return nil
	}

	if cfg.TemplateURL == "" {
		// it will be set in the example docker configs and stuff i'll write, but I don't want to force it on everyone so it defaults to empty.
		log.Warn("template.html not found and template-url is not configured; skipping template download. Configure template-url to enable automatic downloading of a default template.")
		return nil
	}

	log.Info("template.html not found; downloading template", "url", cfg.TemplateURL)

	// Download to a temp file.
	tmpFile, err := os.CreateTemp("", "5000blogs-template-*.zip")
	if err != nil {
		return fmt.Errorf("setup: failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	resp, err := http.Get(cfg.TemplateURL)
	if err != nil {
		return fmt.Errorf("setup: failed to download template from %q: %w", cfg.TemplateURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("setup: template download returned HTTP %d", resp.StatusCode)
	}

	if _, err = io.Copy(tmpFile, resp.Body); err != nil {
		return fmt.Errorf("setup: failed to write template zip: %w", err)
	}
	tmpFile.Close()

	if err := os.MkdirAll(cfg.Paths.Static, 0755); err != nil {
		return fmt.Errorf("setup: failed to create static dir: %w", err)
	}

	// extract zip
	if err := extractZip(tmpFile.Name(), cfg.Paths.Static); err != nil {
		return fmt.Errorf("setup: failed to extract template zip: %w", err)
	}

	log.Info("template extracted successfully", "dest", cfg.Paths.Static)
	return nil
}

func extractZip(src, destDir string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("extractZip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if err := extractZipEntry(f, destDir); err != nil {
			return err
		}
	}
	return nil
}

func extractZipEntry(f *zip.File, destDir string) error {
	// prevent zip-slip.
	target := filepath.Join(destDir, filepath.Clean("/"+f.Name))

	if f.FileInfo().IsDir() {
		return os.MkdirAll(target, f.Mode())
	}

	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return fmt.Errorf("extractZipEntry: mkdir: %w", err)
	}

	dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
	if err != nil {
		return fmt.Errorf("extractZipEntry: create %q: %w", target, err)
	}
	defer dst.Close()

	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("extractZipEntry: open zip entry: %w", err)
	}
	defer rc.Close()

	if _, err = io.Copy(dst, rc); err != nil {
		return fmt.Errorf("extractZipEntry: copy %q: %w", target, err)
	}
	return nil
}
