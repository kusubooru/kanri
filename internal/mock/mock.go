package mock

import (
	"io"

	"github.com/kusubooru/shimmie"
)

type Shimmie struct {
	GetUserFn      func(userID int64) (*shimmie.User, error)
	GetUserInvoked bool

	GetUserByNameFn      func(username string) (*shimmie.User, error)
	GetUserByNameInvoked bool

	FindAliasFn      func(oldTag, newTag string) ([]shimmie.Alias, error)
	FindAliasInvoked bool

	RateImageFn      func(id int, rating string) error
	RateImageInvoked bool

	WriteImageFileFn      func(w io.Writer, path, hash string) error
	WriteImageFileInvoked bool

	GetRatedImagesFn      func(username string) ([]shimmie.RatedImage, error)
	GetRatedImagesInvoked bool

	LogRatingFn      func(imgID int, rating, username, userIP string) error
	LogRatingInvoked bool

	GetContributedTagHistoryFn      func(imageOwnerUsername string) ([]shimmie.ContributedTagHistory, error)
	GetContributedTagHistoryInvoked bool

	GetImageTagHistoryFn      func(imageID int) ([]shimmie.TagHistory, error)
	GetImageTagHistoryInvoked bool

	GetTagHistoryFn      func(imageID int) (*shimmie.TagHistory, error)
	GetTagHistoryInvoked bool
}

func (s *Shimmie) GetUser(userID int64) (*shimmie.User, error) {
	s.GetUserInvoked = true
	return s.GetUserFn(userID)
}
func (s *Shimmie) GetUserByName(username string) (*shimmie.User, error) {
	s.GetUserByNameInvoked = true
	return s.GetUserByNameFn(username)
}
func (s *Shimmie) FindAlias(oldTag, newTag string) ([]shimmie.Alias, error) {
	s.FindAliasInvoked = true
	return s.FindAliasFn(oldTag, newTag)
}
func (s *Shimmie) RateImage(id int, rating string) error {
	s.RateImageInvoked = true
	return s.RateImageFn(id, rating)
}
func (s *Shimmie) WriteImageFile(w io.Writer, path, hash string) error {
	s.WriteImageFileInvoked = true
	return s.WriteImageFileFn(w, path, hash)
}
func (s *Shimmie) GetRatedImages(username string) ([]shimmie.RatedImage, error) {
	s.GetRatedImagesInvoked = true
	return s.GetRatedImagesFn(username)
}
func (s *Shimmie) LogRating(imgID int, rating, username, userIP string) error {
	s.LogRatingInvoked = true
	return s.LogRatingFn(imgID, rating, username, userIP)
}
func (s *Shimmie) GetContributedTagHistory(imageOwnerUsername string) ([]shimmie.ContributedTagHistory, error) {
	s.GetContributedTagHistoryInvoked = true
	return s.GetContributedTagHistoryFn(imageOwnerUsername)
}
func (s *Shimmie) GetImageTagHistory(imageID int) ([]shimmie.TagHistory, error) {
	s.GetImageTagHistoryInvoked = true
	return s.GetImageTagHistoryFn(imageID)
}
func (s *Shimmie) GetTagHistory(imageID int) (*shimmie.TagHistory, error) {
	s.GetTagHistoryInvoked = true
	return s.GetTagHistoryFn(imageID)
}
