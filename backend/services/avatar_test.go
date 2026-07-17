package services

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestBuildAvatarDataAcceptsValidPNG(t *testing.T) {
	var imageData bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	if err := png.Encode(&imageData, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}

	avatar, err := BuildAvatarData(imageData.Bytes())
	if err != nil {
		t.Fatalf("build avatar data: %v", err)
	}
	if !strings.HasPrefix(avatar, "data:image/png;base64,") {
		t.Fatalf("unexpected avatar prefix: %q", avatar[:min(len(avatar), 30)])
	}
}

func TestBuildAvatarDataRejectsInvalidContent(t *testing.T) {
	if _, err := BuildAvatarData([]byte("not an image")); err == nil {
		t.Fatal("invalid avatar content should be rejected")
	}
}

func TestUpdateUserAvatarPersistsInUserRecord(t *testing.T) {
	setupFileTestDB(t)
	const avatar = "data:image/png;base64,dGVzdA=="
	if err := UpdateUserAvatar(2, avatar); err != nil {
		t.Fatalf("update avatar: %v", err)
	}
	user, err := GetUserByID(2)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if user.AvatarData != avatar {
		t.Fatalf("avatar = %q, want %q", user.AvatarData, avatar)
	}
}
