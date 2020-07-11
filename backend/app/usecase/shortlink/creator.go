package shortlink

import (
	"github.com/short-d/app/fw/timer"
	"github.com/short-d/short/backend/app/entity"
	"github.com/short-d/short/backend/app/usecase/keygen"
	"github.com/short-d/short/backend/app/usecase/repository"
	"github.com/short-d/short/backend/app/usecase/risk"
	"github.com/short-d/short/backend/app/usecase/validator"
)

var _ Creator = (*CreatorPersist)(nil)

// ErrAliasExist represents alias unavailable error
type ErrAliasExist string

func (e ErrAliasExist) Error() string {
	return string(e)
}

// ErrInvalidLongLink represents incorrect long link format error
type ErrInvalidLongLink struct {
	LongLink  string
	Violation validator.Violation
}

func (e ErrInvalidLongLink) Error() string {
	return string(e.LongLink)
}

// ErrInvalidCustomAlias represents incorrect custom alias format error
type ErrInvalidCustomAlias struct {
	customAlias string
	Violation   validator.Violation
}

func (e ErrInvalidCustomAlias) Error() string {
	return string(e.customAlias)
}

// ErrMaliciousLongLink represents malicious long link error
type ErrMaliciousLongLink string

func (e ErrMaliciousLongLink) Error() string {
	return string(e)
}

// Creator represents a ShortLink alias creator
type Creator interface {
	CreateShortLink(createArgs entity.ShortLinkInput, user entity.User, isPublic bool) (entity.ShortLink, error)
}

// CreatorPersist represents a ShortLink alias creator which persist the generated
// alias in the repository
type CreatorPersist struct {
	shortLinkRepo     repository.ShortLink
	userShortLinkRepo repository.UserShortLink
	keyGen            keygen.KeyGenerator
	longLinkValidator validator.LongLink
	aliasValidator    validator.CustomAlias
	timer             timer.Timer
	riskDetector      risk.Detector
}

// CreateShortLink persists a new short link with a given or auto generated alias in the repository.
// TODO(issue#235): add functionality for public URLs
func (c CreatorPersist) CreateShortLink(createArgs entity.ShortLinkInput, user entity.User, isPublic bool) (entity.ShortLink, error) {
	longLink := createArgs.GetLongLink("")
	isValid, violation := c.longLinkValidator.IsValid(longLink)
	if !isValid {
		return entity.ShortLink{}, ErrInvalidLongLink{longLink, violation}
	}

	if c.riskDetector.IsURLMalicious(longLink) {
		return entity.ShortLink{}, ErrMaliciousLongLink(longLink)
	}

	customAlias := createArgs.GetCustomAlias("")
	isValid, violation = c.aliasValidator.IsValid(customAlias)
	if !isValid {
		return entity.ShortLink{}, ErrInvalidCustomAlias{customAlias, violation}
	}

	if customAlias == "" {
		autoAlias, err := c.createAutoAlias()
		if err != nil {
			// TODO create error type for fail create auto alias?
			return entity.ShortLink{}, err
		}
		customAlias = autoAlias
	}

	return c.createShortLink(entity.ShortLink{
		LongLink: longLink,
		Alias:    customAlias,
		ExpireAt: createArgs.ExpireAt,
	}, user)
}

func (c CreatorPersist) createAutoAlias() (string, error) {
	key, err := c.keyGen.NewKey()
	if err != nil {
		return "", err
	}
	return string(key), nil
}

func (c CreatorPersist) createShortLink(shortLink entity.ShortLink, user entity.User) (entity.ShortLink, error) {
	isExist, err := c.shortLinkRepo.IsAliasExist(shortLink.Alias)
	if err != nil {
		return entity.ShortLink{}, err
	}

	if isExist {
		return entity.ShortLink{}, ErrAliasExist("short link alias already exist")
	}

	now := c.timer.Now().UTC()
	shortLink.CreatedAt = &now

	err = c.shortLinkRepo.CreateShortLink(shortLink)
	if err != nil {
		return entity.ShortLink{}, err
	}

	err = c.userShortLinkRepo.CreateRelation(user, shortLink)
	return shortLink, err
}

// NewCreatorPersist creates CreatorPersist
func NewCreatorPersist(
	shortLinkRepo repository.ShortLink,
	userShortLinkRepo repository.UserShortLink,
	keyGen keygen.KeyGenerator,
	longLinkValidator validator.LongLink,
	aliasValidator validator.CustomAlias,
	timer timer.Timer,
	riskDetector risk.Detector,
) CreatorPersist {
	return CreatorPersist{
		shortLinkRepo:     shortLinkRepo,
		userShortLinkRepo: userShortLinkRepo,
		keyGen:            keyGen,
		longLinkValidator: longLinkValidator,
		aliasValidator:    aliasValidator,
		timer:             timer,
		riskDetector:      riskDetector,
	}
}
