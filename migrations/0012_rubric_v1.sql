-- +goose Up
UPDATE taste_profiles SET rubric_version = 'v1' WHERE rubric_version = 'spike-s01';

-- +goose Down
UPDATE taste_profiles SET rubric_version = 'spike-s01' WHERE rubric_version = 'v1';
