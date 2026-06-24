package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveImagePath(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup directory structure
	postDir := filepath.Join(tmpDir, "posts")
	configImageDir := filepath.Join(tmpDir, "images")
	cwd, _ := os.Getwd()

	err := os.MkdirAll(postDir, 0755)
	if err != nil {
		t.Fatalf("failed to create postDir: %v", err)
	}
	err = os.MkdirAll(configImageDir, 0755)
	if err != nil {
		t.Fatalf("failed to create configImageDir: %v", err)
	}

	// Create dummy image files
	postDirImg := filepath.Join(postDir, "pic1.png")
	_ = os.WriteFile(postDirImg, []byte("postDir-img"), 0644)

	cwdImg := filepath.Join(cwd, "pic2.png")
	_ = os.WriteFile(cwdImg, []byte("cwd-img"), 0644)
	defer os.Remove(cwdImg)

	configImg := filepath.Join(configImageDir, "pic3.png")
	_ = os.WriteFile(configImg, []byte("config-img"), 0644)

	tests := []struct {
		name           string
		postDir        string
		imagePath      string
		configImageDir string
		expectedExist  bool
		expectedPath   string
	}{
		{
			name:           "Absolute path exists",
			postDir:        postDir,
			imagePath:      postDirImg,
			configImageDir: configImageDir,
			expectedExist:  true,
			expectedPath:   postDirImg,
		},
		{
			name:           "Relative to post directory",
			postDir:        postDir,
			imagePath:      "pic1.png",
			configImageDir: configImageDir,
			expectedExist:  true,
			expectedPath:   postDirImg,
		},
		{
			name:           "Relative to current working directory",
			postDir:        postDir,
			imagePath:      "pic2.png",
			configImageDir: configImageDir,
			expectedExist:  true,
			expectedPath:   cwdImg,
		},
		{
			name:           "Relative to configured image directory",
			postDir:        postDir,
			imagePath:      "pic3.png",
			configImageDir: configImageDir,
			expectedExist:  true,
			expectedPath:   configImg,
		},
		{
			name:           "Non-existent image",
			postDir:        postDir,
			imagePath:      "does_not_exist.png",
			configImageDir: configImageDir,
			expectedExist:  false,
			expectedPath:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, ok := resolveImagePath(tt.postDir, tt.imagePath, tt.configImageDir)
			if ok != tt.expectedExist {
				t.Errorf("expected exists=%v, got %v", tt.expectedExist, ok)
			}
			if ok {
				absExpected, _ := filepath.Abs(tt.expectedPath)
				absActual, _ := filepath.Abs(path)
				if absActual != absExpected {
					t.Errorf("expected path %q, got %q", absExpected, absActual)
				}
			}
		})
	}
}
