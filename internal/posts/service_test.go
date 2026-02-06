package posts

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jeremyjsx/entries/internal/storage"
)

type mockRepo struct {
	create   func(ctx context.Context, title, slug, s3Key string) (*Post, error)
	getBySlug func(ctx context.Context, slug string) (*Post, error)
	list     func(ctx context.Context, params ListParams) ([]*Post, error)
	count    func(ctx context.Context, status *Status) (int64, error)
	update   func(ctx context.Context, id uuid.UUID, title, slug, s3Key string) (*Post, error)
	delete   func(ctx context.Context, slug string) error
	publish  func(ctx context.Context, slug string) (*Post, error)
}

func (m *mockRepo) Create(ctx context.Context, title, slug, s3Key string) (*Post, error) {
	if m.create != nil {
		return m.create(ctx, title, slug, s3Key)
	}
	return nil, nil
}

func (m *mockRepo) GetBySlug(ctx context.Context, slug string) (*Post, error) {
	if m.getBySlug != nil {
		return m.getBySlug(ctx, slug)
	}
	return nil, ErrNotFound
}

func (m *mockRepo) List(ctx context.Context, params ListParams) ([]*Post, error) {
	if m.list != nil {
		return m.list(ctx, params)
	}
	return nil, nil
}

func (m *mockRepo) Count(ctx context.Context, status *Status) (int64, error) {
	if m.count != nil {
		return m.count(ctx, status)
	}
	return 0, nil
}

func (m *mockRepo) Update(ctx context.Context, id uuid.UUID, title, slug, s3Key string) (*Post, error) {
	if m.update != nil {
		return m.update(ctx, id, title, slug, s3Key)
	}
	return nil, nil
}

func (m *mockRepo) Delete(ctx context.Context, slug string) error {
	if m.delete != nil {
		return m.delete(ctx, slug)
	}
	return nil
}

func (m *mockRepo) Publish(ctx context.Context, slug string) (*Post, error) {
	if m.publish != nil {
		return m.publish(ctx, slug)
	}
	return nil, nil
}

type mockStorage struct {
	upload      func(ctx context.Context, key string, body io.Reader, contentType string) error
	download    func(ctx context.Context, key string) (io.ReadCloser, error)
	delete      func(ctx context.Context, key string) error
	deletePrefix func(ctx context.Context, prefix string) error
	exists      func(ctx context.Context, key string) (bool, error)
}

func (m *mockStorage) Upload(ctx context.Context, key string, body io.Reader, contentType string) error {
	if m.upload != nil {
		return m.upload(ctx, key, body, contentType)
	}
	return nil
}

func (m *mockStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	if m.download != nil {
		return m.download(ctx, key)
	}
	return nil, storage.ErrNotFound
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	if m.delete != nil {
		return m.delete(ctx, key)
	}
	return nil
}

func (m *mockStorage) DeletePrefix(ctx context.Context, prefix string) error {
	if m.deletePrefix != nil {
		return m.deletePrefix(ctx, prefix)
	}
	return nil
}

func (m *mockStorage) Exists(ctx context.Context, key string) (bool, error) {
	if m.exists != nil {
		return m.exists(ctx, key)
	}
	return false, nil
}

func mustUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		panic(err)
	}
	return id
}

func TestService_CreatePost(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		want := &Post{ID: mustUUID("00000000-0000-0000-0000-000000000001"), Title: "Hi", Slug: "hi", S3Key: "posts/hi.md", Status: Draft}
		repo := &mockRepo{
			create: func(ctx context.Context, title, slug, s3Key string) (*Post, error) {
				if title != "Hi" || slug != "hi" || s3Key != "posts/hi.md" {
					t.Errorf("Create got title=%q slug=%q s3Key=%q", title, slug, s3Key)
				}
				return want, nil
			},
		}
		var uploadBody []byte
		st := &mockStorage{
			upload: func(ctx context.Context, key string, body io.Reader, contentType string) error {
				var err error
				uploadBody, err = io.ReadAll(body)
				if err != nil {
					return err
				}
				if key != "posts/hi.md" || contentType != "text/markdown" {
					t.Errorf("Upload key=%q contentType=%q", key, contentType)
				}
				return nil
			},
		}
		svc := NewService(repo, st, "b", "us-east-1", "")
		got, err := svc.CreatePost(ctx, "Hi", "hi", "# Hello")
		if err != nil {
			t.Fatalf("CreatePost: %v", err)
		}
		if got.ID != want.ID || got.Slug != want.Slug {
			t.Errorf("got %+v", got)
		}
		if !bytes.Equal(uploadBody, []byte("# Hello")) {
			t.Errorf("upload body = %q", uploadBody)
		}
	})

	t.Run("repo returns ErrSlugExists", func(t *testing.T) {
		ctx := context.Background()
		repo := &mockRepo{create: func(context.Context, string, string, string) (*Post, error) { return nil, ErrSlugExists }}
		svc := NewService(repo, &mockStorage{}, "b", "r", "")
		_, err := svc.CreatePost(ctx, "T", "t", "body")
		if !errors.Is(err, ErrSlugExists) {
			t.Errorf("got err %v", err)
		}
	})

	t.Run("storage upload fails", func(t *testing.T) {
		ctx := context.Background()
		repo := &mockRepo{create: func(context.Context, string, string, string) (*Post, error) {
			return &Post{ID: uuid.New(), Slug: "x"}, nil
		}}
		st := &mockStorage{upload: func(context.Context, string, io.Reader, string) error {
			return errors.New("upload failed")
		}}
		svc := NewService(repo, st, "b", "r", "")
		_, err := svc.CreatePost(ctx, "T", "x", "body")
		if err == nil || !strings.Contains(err.Error(), "upload to s3") {
			t.Errorf("got err %v", err)
		}
	})
}

func TestService_GetPostBySlug(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		want := &Post{ID: uuid.New(), Title: "A", Slug: "a", Status: Draft}
		repo := &mockRepo{getBySlug: func(context.Context, string) (*Post, error) { return want, nil }}
		svc := NewService(repo, &mockStorage{}, "b", "r", "")
		got, err := svc.GetPostBySlug(ctx, "a")
		if err != nil {
			t.Fatalf("GetPostBySlug: %v", err)
		}
		if got != want {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctx := context.Background()
		repo := &mockRepo{getBySlug: func(context.Context, string) (*Post, error) { return nil, ErrNotFound }}
		svc := NewService(repo, &mockStorage{}, "b", "r", "")
		_, err := svc.GetPostBySlug(ctx, "missing")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("got err %v", err)
		}
	})
}

func TestService_GetPostContent(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		repo := &mockRepo{getBySlug: func(context.Context, string) (*Post, error) {
			return &Post{Slug: "a", S3Key: "posts/a.md"}, nil
		}}
		st := &mockStorage{download: func(context.Context, string) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("markdown here")), nil
		}}
		svc := NewService(repo, st, "b", "r", "")
		body, err := svc.GetPostContent(ctx, "a")
		if err != nil {
			t.Fatalf("GetPostContent: %v", err)
		}
		if string(body) != "markdown here" {
			t.Errorf("got body %q", body)
		}
	})

	t.Run("post not found", func(t *testing.T) {
		ctx := context.Background()
		repo := &mockRepo{getBySlug: func(context.Context, string) (*Post, error) { return nil, ErrNotFound }}
		svc := NewService(repo, &mockStorage{}, "b", "r", "")
		_, err := svc.GetPostContent(ctx, "x")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("got err %v", err)
		}
	})

	t.Run("storage ErrNotFound", func(t *testing.T) {
		ctx := context.Background()
		repo := &mockRepo{getBySlug: func(context.Context, string) (*Post, error) {
			return &Post{S3Key: "posts/a.md"}, nil
		}}
		st := &mockStorage{download: func(context.Context, string) (io.ReadCloser, error) {
			return nil, storage.ErrNotFound
		}}
		svc := NewService(repo, st, "b", "r", "")
		_, err := svc.GetPostContent(ctx, "a")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("got err %v", err)
		}
	})
}

func TestService_ListPosts(t *testing.T) {
	t.Run("success and pagination", func(t *testing.T) {
		ctx := context.Background()
		posts := []*Post{{ID: uuid.New(), Slug: "one"}}
		repo := &mockRepo{
			list:  func(context.Context, ListParams) ([]*Post, error) { return posts, nil },
			count: func(context.Context, *Status) (int64, error) { return 1, nil },
		}
		svc := NewService(repo, &mockStorage{}, "b", "r", "")
		result, err := svc.ListPosts(ctx, 1, 10, nil)
		if err != nil {
			t.Fatalf("ListPosts: %v", err)
		}
		if result.Total != 1 || result.Page != 1 || result.PerPage != 10 || len(result.Posts) != 1 {
			t.Errorf("got %+v", result)
		}
	})

	t.Run("page and per_page normalized", func(t *testing.T) {
		ctx := context.Background()
		repo := &mockRepo{
			list: func(_ context.Context, p ListParams) ([]*Post, error) {
				if p.Limit != 20 || p.Offset != 0 {
					t.Errorf("ListParams Limit=%d Offset=%d", p.Limit, p.Offset)
				}
				return nil, nil
			},
			count: func(context.Context, *Status) (int64, error) { return 0, nil },
		}
		svc := NewService(repo, &mockStorage{}, "b", "r", "")
		result, err := svc.ListPosts(ctx, 0, 0, nil)
		if err != nil {
			t.Fatalf("ListPosts: %v", err)
		}
		if result.Page != 1 || result.PerPage != 20 {
			t.Errorf("got page=%d perPage=%d", result.Page, result.PerPage)
		}
	})
}

func TestService_UpdatePost(t *testing.T) {
	postID := mustUUID("10000000-0000-0000-0000-000000000001")
	existing := &Post{ID: postID, Title: "Old", Slug: "old", S3Key: "posts/old.md"}

	t.Run("success title only", func(t *testing.T) {
		ctx := context.Background()
		title := "New Title"
		repo := &mockRepo{
			getBySlug: func(context.Context, string) (*Post, error) { return existing, nil },
			update: func(ctx context.Context, id uuid.UUID, title, slug, s3Key string) (*Post, error) {
				if title != "New Title" || slug != "old" || s3Key != "posts/old.md" {
					t.Errorf("Update got title=%q slug=%q s3Key=%q", title, slug, s3Key)
				}
				return &Post{ID: id, Title: title, Slug: slug, S3Key: s3Key}, nil
			},
		}
		svc := NewService(repo, &mockStorage{}, "b", "r", "")
		got, err := svc.UpdatePost(ctx, "old", &title, nil, nil)
		if err != nil {
			t.Fatalf("UpdatePost: %v", err)
		}
		if got.Title != "New Title" {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctx := context.Background()
		repo := &mockRepo{getBySlug: func(context.Context, string) (*Post, error) { return nil, ErrNotFound }}
		svc := NewService(repo, &mockStorage{}, "b", "r", "")
		title := "X"
		_, err := svc.UpdatePost(ctx, "x", &title, nil, nil)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("got err %v", err)
		}
	})

	t.Run("success with content and new slug", func(t *testing.T) {
		ctx := context.Background()
		newTitle := "New"
		newSlug := "new-slug"
		newContent := "body"
		var uploadedKey string
		var uploadedContent []byte
		repo := &mockRepo{
			getBySlug: func(context.Context, string) (*Post, error) { return existing, nil },
			update: func(ctx context.Context, id uuid.UUID, title, slug, s3Key string) (*Post, error) {
				if title != "New" || slug != "new-slug" || s3Key != "posts/new-slug.md" {
					t.Errorf("Update got title=%q slug=%q s3Key=%q", title, slug, s3Key)
				}
				return &Post{ID: id, Title: title, Slug: slug, S3Key: s3Key}, nil
			},
		}
		st := &mockStorage{
			upload: func(ctx context.Context, key string, body io.Reader, contentType string) error {
				uploadedKey = key
				uploadedContent, _ = io.ReadAll(body)
				return nil
			},
			delete: func(context.Context, string) error { return nil },
		}
		svc := NewService(repo, st, "b", "r", "")
		got, err := svc.UpdatePost(ctx, "old", &newTitle, &newSlug, &newContent)
		if err != nil {
			t.Fatalf("UpdatePost: %v", err)
		}
		if got.Slug != "new-slug" {
			t.Errorf("got slug %q", got.Slug)
		}
		if uploadedKey != "posts/new-slug.md" || string(uploadedContent) != "body" {
			t.Errorf("upload key=%q content=%q", uploadedKey, uploadedContent)
		}
	})

	t.Run("content upload fails", func(t *testing.T) {
		ctx := context.Background()
		content := "x"
		repo := &mockRepo{getBySlug: func(context.Context, string) (*Post, error) { return existing, nil }}
		st := &mockStorage{upload: func(context.Context, string, io.Reader, string) error {
			return errors.New("upload failed")
		}}
		svc := NewService(repo, st, "b", "r", "")
		_, err := svc.UpdatePost(ctx, "old", nil, nil, &content)
		if err == nil || !strings.Contains(err.Error(), "upload to s3") {
			t.Errorf("got err %v", err)
		}
	})

	t.Run("no content, slug change (move content)", func(t *testing.T) {
		ctx := context.Background()
		newSlug := "new-slug"
		downloaded := []byte("existing markdown")
		var uploadedKey string
		repo := &mockRepo{
			getBySlug: func(context.Context, string) (*Post, error) { return existing, nil },
			update: func(ctx context.Context, id uuid.UUID, title, slug, s3Key string) (*Post, error) {
				if s3Key != "posts/new-slug.md" {
					t.Errorf("Update s3Key=%q", s3Key)
				}
				return &Post{ID: id, Slug: slug, S3Key: s3Key}, nil
			},
		}
		st := &mockStorage{
			download: func(ctx context.Context, key string) (io.ReadCloser, error) {
				if key != "posts/old.md" {
					t.Errorf("Download key=%q", key)
				}
				return io.NopCloser(bytes.NewReader(downloaded)), nil
			},
			upload: func(ctx context.Context, key string, body io.Reader, contentType string) error {
				uploadedKey = key
				return nil
			},
			delete: func(context.Context, string) error { return nil },
		}
		svc := NewService(repo, st, "b", "r", "")
		got, err := svc.UpdatePost(ctx, "old", nil, &newSlug, nil)
		if err != nil {
			t.Fatalf("UpdatePost: %v", err)
		}
		if got.S3Key != "posts/new-slug.md" {
			t.Errorf("got S3Key %q", got.S3Key)
		}
		if uploadedKey != "posts/new-slug.md" {
			t.Errorf("uploaded to %q", uploadedKey)
		}
	})

	t.Run("no content, slug change - download fails", func(t *testing.T) {
		ctx := context.Background()
		newSlug := "new-slug"
		repo := &mockRepo{getBySlug: func(context.Context, string) (*Post, error) { return existing, nil }}
		st := &mockStorage{download: func(context.Context, string) (io.ReadCloser, error) {
			return nil, errors.New("download failed")
		}}
		svc := NewService(repo, st, "b", "r", "")
		_, err := svc.UpdatePost(ctx, "old", nil, &newSlug, nil)
		if err == nil || !strings.Contains(err.Error(), "download current content") {
			t.Errorf("got err %v", err)
		}
	})

	t.Run("no content, slug same - s3Key unchanged", func(t *testing.T) {
		ctx := context.Background()
		repo := &mockRepo{
			getBySlug: func(context.Context, string) (*Post, error) { return existing, nil },
			update: func(ctx context.Context, id uuid.UUID, title, slug, s3Key string) (*Post, error) {
				if s3Key != "posts/old.md" {
					t.Errorf("expected s3Key posts/old.md, got %q", s3Key)
				}
				return &Post{S3Key: s3Key}, nil
			},
		}
		svc := NewService(repo, &mockStorage{}, "b", "r", "")
		got, err := svc.UpdatePost(ctx, "old", nil, nil, nil)
		if err != nil {
			t.Fatalf("UpdatePost: %v", err)
		}
		if got.S3Key != "posts/old.md" {
			t.Errorf("got S3Key %q", got.S3Key)
		}
	})

	t.Run("repo Update returns ErrSlugExists", func(t *testing.T) {
		ctx := context.Background()
		title := "X"
		repo := &mockRepo{
			getBySlug: func(context.Context, string) (*Post, error) { return existing, nil },
			update:    func(context.Context, uuid.UUID, string, string, string) (*Post, error) { return nil, ErrSlugExists },
		}
		svc := NewService(repo, &mockStorage{}, "b", "r", "")
		_, err := svc.UpdatePost(ctx, "old", &title, nil, nil)
		if !errors.Is(err, ErrSlugExists) {
			t.Errorf("got err %v", err)
		}
	})
}

func TestService_DeletePost(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		repo := &mockRepo{
			getBySlug: func(context.Context, string) (*Post, error) {
				return &Post{Slug: "a", S3Key: "posts/a.md"}, nil
			},
			delete: func(context.Context, string) error { return nil },
		}
		st := &mockStorage{
			delete:      func(context.Context, string) error { return nil },
			deletePrefix: func(context.Context, string) error { return nil },
		}
		svc := NewService(repo, st, "b", "r", "")
		err := svc.DeletePost(ctx, "a")
		if err != nil {
			t.Fatalf("DeletePost: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctx := context.Background()
		repo := &mockRepo{getBySlug: func(context.Context, string) (*Post, error) { return nil, ErrNotFound }}
		svc := NewService(repo, &mockStorage{}, "b", "r", "")
		err := svc.DeletePost(ctx, "x")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("got err %v", err)
		}
	})
}

func TestService_PublishPost(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		want := &Post{ID: uuid.New(), Slug: "p", Status: Published}
		repo := &mockRepo{publish: func(context.Context, string) (*Post, error) { return want, nil }}
		svc := NewService(repo, &mockStorage{}, "b", "r", "")
		got, err := svc.PublishPost(ctx, "p")
		if err != nil {
			t.Fatalf("PublishPost: %v", err)
		}
		if got != want {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctx := context.Background()
		repo := &mockRepo{publish: func(context.Context, string) (*Post, error) { return nil, ErrNotFound }}
		svc := NewService(repo, &mockStorage{}, "b", "r", "")
		_, err := svc.PublishPost(ctx, "x")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("got err %v", err)
		}
	})
}

func TestService_s3PublicURL(t *testing.T) {
	repo := &mockRepo{}
	st := &mockStorage{}
	svc := NewService(repo, st, "mybucket", "us-east-1", "")
	u := svc.s3PublicURL("posts/a.md")
	if u != "https://mybucket.s3.us-east-1.amazonaws.com/posts/a.md" {
		t.Errorf("got %q", u)
	}
	svc2 := NewService(repo, st, "b", "r", "https://cdn.example.com")
	u2 := svc2.s3PublicURL("posts/a.md")
	if u2 != "https://cdn.example.com/posts/a.md" {
		t.Errorf("got %q", u2)
	}
}

func TestService_processMarkdownImages(t *testing.T) {
	ctx := context.Background()
	uploaded := make(map[string][]byte)
	repo := &mockRepo{create: func(context.Context, string, string, string) (*Post, error) {
		return &Post{ID: uuid.New(), Slug: "img"}, nil
	}}
	st := &mockStorage{
		upload: func(ctx context.Context, key string, body io.Reader, contentType string) error {
			data, _ := io.ReadAll(body)
			uploaded[key] = data
			return nil
		},
	}
	svc := NewService(repo, st, "b", "r", "")
	b64 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
	content := "# Post\n\n![alt](data:image/png;base64," + b64 + ")"
	_, err := svc.CreatePost(ctx, "Img", "img", content)
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}
	if _, ok := uploaded["posts/img.md"]; !ok {
		t.Errorf("expected posts/img.md upload")
	}
	markdown := string(uploaded["posts/img.md"])
	if strings.Contains(markdown, "data:image") {
		t.Errorf("expected data URL to be replaced with S3 URL, got %s", markdown)
	}
}

func TestService_processMarkdownImages_disallowedType(t *testing.T) {
	ctx := context.Background()
	uploaded := make(map[string][]byte)
	repo := &mockRepo{create: func(context.Context, string, string, string) (*Post, error) {
		return &Post{ID: uuid.New(), Slug: "img"}, nil
	}}
	st := &mockStorage{
		upload: func(ctx context.Context, key string, body io.Reader, contentType string) error {
			data, _ := io.ReadAll(body)
			uploaded[key] = data
			return nil
		},
	}
	svc := NewService(repo, st, "b", "r", "")
	content := "# Post\n\n![alt](data:image/svg+xml;base64,PHN2Zy8+)"
	_, err := svc.CreatePost(ctx, "Img", "img", content)
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}
	markdown := string(uploaded["posts/img.md"])
	if !strings.Contains(markdown, "data:image/svg+xml") {
		t.Errorf("disallowed type should be left as data URL: %s", markdown)
	}
}

func TestService_processMarkdownImages_invalidBase64(t *testing.T) {
	ctx := context.Background()
	uploaded := make(map[string][]byte)
	repo := &mockRepo{create: func(context.Context, string, string, string) (*Post, error) {
		return &Post{ID: uuid.New(), Slug: "img"}, nil
	}}
	st := &mockStorage{
		upload: func(ctx context.Context, key string, body io.Reader, contentType string) error {
			data, _ := io.ReadAll(body)
			uploaded[key] = data
			return nil
		},
	}
	svc := NewService(repo, st, "b", "r", "")
	content := "# Post\n\n![alt](data:image/png;base64,not-valid-base64!!)"
	_, err := svc.CreatePost(ctx, "Img", "img", content)
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}
	markdown := string(uploaded["posts/img.md"])
	if !strings.Contains(markdown, "data:image/png") {
		t.Errorf("invalid base64 should leave data URL: %s", markdown)
	}
}

func TestService_processMarkdownImages_uploadFails(t *testing.T) {
	ctx := context.Background()
	uploadCount := 0
	repo := &mockRepo{create: func(context.Context, string, string, string) (*Post, error) {
		return &Post{ID: uuid.New(), Slug: "img"}, nil
	}}
	st := &mockStorage{
		upload: func(ctx context.Context, key string, body io.Reader, contentType string) error {
			uploadCount++
			if strings.HasPrefix(key, "posts/img/images/") {
				return errors.New("image upload failed")
			}
			return nil
		},
	}
	svc := NewService(repo, st, "b", "r", "")
	b64 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
	content := "# Post\n\n![alt](data:image/png;base64," + b64 + ")"
	_, err := svc.CreatePost(ctx, "Img", "img", content)
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}
	if uploadCount < 2 {
		t.Errorf("expected at least 2 uploads (content + image attempt), got %d", uploadCount)
	}
}
