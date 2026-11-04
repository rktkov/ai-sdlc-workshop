package library

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRepo struct {
	items   map[string]Entry
	saved   *Entry
	marked  bool
	removed bool
	err     error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{items: map[string]Entry{}}
}

func key(userID, videoID string) string { return userID + "|" + videoID }

func (f *fakeRepo) Get(_ context.Context, userID, videoID string) (Entry, error) {
	if f.err != nil {
		return Entry{}, f.err
	}
	entry, ok := f.items[key(userID, videoID)]
	if !ok {
		return Entry{}, ErrNotFound
	}
	return entry, nil
}

func (f *fakeRepo) List(context.Context, string) ([]Entry, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([]Entry, 0, len(f.items))
	for _, entry := range f.items {
		out = append(out, entry)
	}
	return out, nil
}

func (f *fakeRepo) Save(_ context.Context, entry Entry) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.items[key(entry.UserID, entry.VideoID)]; ok {
		return ErrExists
	}
	f.items[key(entry.UserID, entry.VideoID)] = entry
	f.saved = &entry
	return nil
}

func (f *fakeRepo) MarkDownloaded(_ context.Context, userID, videoID, extension string, bytes int64, now time.Time) (Entry, error) {
	if f.err != nil {
		return Entry{}, f.err
	}
	entry, ok := f.items[key(userID, videoID)]
	if !ok {
		return Entry{}, ErrNotFound
	}
	entry.State = StateDownloaded
	entry.FileExtension = extension
	entry.FileBytes = bytes
	entry.DownloadedAt = now
	entry.UpdatedAt = now
	f.items[key(userID, videoID)] = entry
	f.marked = true
	return entry, nil
}

func (f *fakeRepo) Remove(_ context.Context, userID, videoID string) error {
	if f.err != nil {
		return f.err
	}
	delete(f.items, key(userID, videoID))
	f.removed = true
	return nil
}

type fakeFiles struct {
	bytes     int64
	statErr   error
	deleteErr error
	dir       string
	path      string
}

func (f *fakeFiles) Dir() string { return f.dir }
func (f *fakeFiles) Path(videoID, extension string) string {
	return f.dir + "/" + videoID + "." + extension
}
func (f *fakeFiles) Stat(_, _ string) (int64, error) {
	return f.bytes, f.statErr
}
func (f *fakeFiles) Delete(_, _ string) error { return f.deleteErr }

type fakeDownloader struct {
	result DownloadResult
	err    error
	dir    string
	calls  int
}

func (f *fakeDownloader) Download(_ context.Context, _, dir string) (DownloadResult, error) {
	f.calls++
	f.dir = dir
	return f.result, f.err
}

func savedEntry() Entry {
	return Entry{
		UserID: "owner", VideoID: "v1", State: StateSaved, Title: "Title",
		ChannelID: "ch1", DurationSeconds: 90,
	}
}

func TestEntryValidate(t *testing.T) {
	if err := savedEntry().Validate(); err != nil {
		t.Fatalf("Validate saved: %v", err)
	}
	downloaded := savedEntry()
	downloaded.State = StateDownloaded
	downloaded.FileExtension = "mp4"
	downloaded.FileBytes = 10
	downloaded.DownloadedAt = time.Now().UTC()
	if err := downloaded.Validate(); err != nil {
		t.Fatalf("Validate downloaded: %v", err)
	}
	tests := []struct {
		name string
		edit func(*Entry)
	}{
		{name: "no user", edit: func(e *Entry) { e.UserID = "" }},
		{name: "bad video", edit: func(e *Entry) { e.VideoID = "../x" }},
		{name: "no title", edit: func(e *Entry) { e.Title = "" }},
		{name: "negative duration", edit: func(e *Entry) { e.DurationSeconds = -1 }},
		{name: "saved with file", edit: func(e *Entry) { e.FileExtension = "mp4" }},
		{name: "bad state", edit: func(e *Entry) { e.State = "bad" }},
		{name: "downloaded without file", edit: func(e *Entry) { e.State = StateDownloaded }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := savedEntry()
			tt.edit(&entry)
			if err := entry.Validate(); err == nil {
				t.Fatal("Validate = nil, want an error")
			}
		})
	}
}

func TestSaveHappy(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	service := &Service{Repo: repo, Now: func() time.Time { return now }}
	entry, err := service.Save(context.Background(), savedEntry())
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if entry.State != StateSaved || !entry.SavedAt.Equal(now) || repo.saved == nil {
		t.Fatalf("entry = %+v", entry)
	}
}

func TestSaveDuplicate(t *testing.T) {
	repo := newFakeRepo()
	service := &Service{Repo: repo}
	if _, err := service.Save(context.Background(), savedEntry()); err != nil {
		t.Fatalf("Save first: %v", err)
	}
	entry, err := service.Save(context.Background(), savedEntry())
	if err != nil {
		t.Fatalf("Save duplicate: %v", err)
	}
	if entry.VideoID != "v1" {
		t.Fatalf("entry = %+v", entry)
	}
}

func TestDownloadHappy(t *testing.T) {
	repo := newFakeRepo()
	files := &fakeFiles{bytes: 42, dir: "/library"}
	downloader := &fakeDownloader{result: DownloadResult{Extension: "mp4"}}
	service := &Service{Repo: repo, Files: files, Downloader: downloader}
	if _, err := service.Save(context.Background(), savedEntry()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	entry, err := service.Download(context.Background(), "owner", "v1")
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if entry.State != StateDownloaded || entry.FileExtension != "mp4" || entry.FileBytes != 42 {
		t.Fatalf("entry = %+v", entry)
	}
	if downloader.dir != "/library" || !repo.marked {
		t.Fatalf("downloader dir = %q, marked = %t", downloader.dir, repo.marked)
	}
}

func TestDownloadUnhappy(t *testing.T) {
	repo := newFakeRepo()
	service := &Service{Repo: repo, Files: &fakeFiles{bytes: 42}, Downloader: &fakeDownloader{err: errors.New("down")}}
	if _, err := service.Save(context.Background(), savedEntry()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := service.Download(context.Background(), "owner", "v1"); err == nil {
		t.Fatal("Download(failure) = nil, want an error")
	}
	entry, _ := repo.Get(context.Background(), "owner", "v1")
	if entry.State != StateSaved {
		t.Fatalf("entry = %+v", entry)
	}

	service.Downloader = &fakeDownloader{result: DownloadResult{Extension: "mp4"}}
	service.Files = &fakeFiles{statErr: errors.New("missing")}
	if _, err := service.Download(context.Background(), "owner", "v1"); err == nil {
		t.Fatal("Download(missing file) = nil, want an error")
	}
}

func TestDownloadAlreadyDownloaded(t *testing.T) {
	repo := newFakeRepo()
	entry := savedEntry()
	entry.State = StateDownloaded
	entry.FileExtension = "mp4"
	entry.FileBytes = 10
	entry.DownloadedAt = time.Now().UTC()
	repo.items[key("owner", "v1")] = entry
	downloader := &fakeDownloader{result: DownloadResult{Extension: "mp4"}}
	service := &Service{Repo: repo, Files: &fakeFiles{bytes: 10}, Downloader: downloader}
	if _, err := service.Download(context.Background(), "owner", "v1"); err != nil {
		t.Fatalf("Download: %v", err)
	}
	if downloader.calls != 0 {
		t.Fatal("already downloaded entry was downloaded again")
	}
}

func TestRemove(t *testing.T) {
	repo := newFakeRepo()
	files := &fakeFiles{}
	service := &Service{Repo: repo, Files: files}
	if _, err := service.Save(context.Background(), savedEntry()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := service.Remove(context.Background(), "owner", "v1"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if !repo.removed {
		t.Fatal("entry was not removed")
	}
	if err := service.Remove(context.Background(), "owner", "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Remove(missing) = %v, want ErrNotFound", err)
	}
}

func TestList(t *testing.T) {
	repo := newFakeRepo()
	service := &Service{Repo: repo}
	if _, err := service.Save(context.Background(), savedEntry()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	list, err := service.List(context.Background(), "owner")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list = %+v", list)
	}
}
