package gold

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalize(t *testing.T) {
	const normalized = "{\n  \"sender\": {\n    \"id\": 866677,\n    \"login\": \"ernado\",\n    \"display_login\": \"ernado\",\n    \"gravatar_id\": \"\",\n    \"url\": \"https://api.github.com/users/ernado\",\n    \"html_url\": \"https://github.com/ernado\",\n    \"avatar_url\": \"https://avatars.githubusercontent.com/u/866677?\"\n  },\n  \"action\": \"opened\",\n  \"issue\": {\n      \"url\": \"https://api.github.com/repos/ernado/oss-estimator/issues/14\",\n      \"repository_url\": \"https://api.github.com/repos/ernado/oss-estimator\",\n      \"labels_url\": \"https://api.github.com/repos/ernado/oss-estimator/issues/14/labels{/name}\",\n      \"comments_url\": \"https://api.github.com/repos/ernado/oss-estimator/issues/14/comments\",\n      \"events_url\": \"https://api.github.com/repos/ernado/oss-estimator/issues/14/events\",\n      \"html_url\": \"https://github.com/ernado/oss-estimator/issues/14\",\n      \"id\": 1637789355,\n      \"node_id\": \"I_kwDOJGfUlc5hnq6r\",\n      \"number\": 14,\n      \"title\": \"test4\",\n      \"user\": {\n        \"login\": \"ernado\",\n        \"id\": 866677,\n        \"node_id\": \"MDQ6VXNlcjg2NjY3Nw==\",\n        \"avatar_url\": \"https://avatars.githubusercontent.com/u/866677?v=4\",\n        \"gravatar_id\": \"\",\n        \"url\": \"https://api.github.com/users/ernado\",\n        \"html_url\": \"https://github.com/ernado\",\n        \"followers_url\": \"https://api.github.com/users/ernado/followers\",\n        \"following_url\": \"https://api.github.com/users/ernado/following{/other_user}\",\n        \"gists_url\": \"https://api.github.com/users/ernado/gists{/gist_id}\",\n        \"starred_url\": \"https://api.github.com/users/ernado/starred{/owner}{/repo}\",\n        \"subscriptions_url\": \"https://api.github.com/users/ernado/subscriptions\",\n        \"organizations_url\": \"https://api.github.com/users/ernado/orgs\",\n        \"repos_url\": \"https://api.github.com/users/ernado/repos\",\n        \"events_url\": \"https://api.github.com/users/ernado/events{/privacy}\",\n        \"received_events_url\": \"https://api.github.com/users/ernado/received_events\",\n        \"type\": \"User\",\n        \"site_admin\": false\n      },\n      \"labels\": [],\n      \"state\": \"open\",\n      \"locked\": false,\n      \"assignee\": null,\n      \"assignees\": [],\n      \"milestone\": null,\n      \"comments\": 0,\n      \"created_at\": \"2023-03-23T15:41:09Z\",\n      \"updated_at\": \"2023-03-23T15:41:09Z\",\n      \"closed_at\": null,\n      \"author_association\": \"OWNER\",\n      \"active_lock_reason\": null,\n      \"body\": null,\n      \"reactions\": {\n        \"url\": \"https://api.github.com/repos/ernado/oss-estimator/issues/14/reactions\",\n        \"total_count\": 0,\n        \"+1\": 0,\n        \"-1\": 0,\n        \"laugh\": 0,\n        \"hooray\": 0,\n        \"confused\": 0,\n        \"heart\": 0,\n        \"rocket\": 0,\n        \"eyes\": 0\n      },\n      \"timeline_url\": \"https://api.github.com/repos/ernado/oss-estimator/issues/14/timeline\",\n      \"performed_via_github_app\": null,\n      \"state_reason\": null\n    },\n  \"repository\": {\n    \"id\": 610784405,\n    \"full_name\": \"ernado/oss-estimator\",\n    \"url\": \"https://api.github.com/repos/ernado/oss-estimator\",\n    \"html_url\": \"https://github.com/ernado/oss-estimator\",\n    \"name\": \"oss-estimator\",\n    \"owner\": {\n      \"login\": \"ernado\"\n    }\n  }\n}"
	const raw = "{\n  \"sender\": {\n    \"id\": 866677,\n    \"login\": \"ernado\",\n    \"display_login\": \"ernado\",\n    \"gravatar_id\": \"\",\n    \"url\": \"https://api.github.com/users/ernado\",\n    \"html_url\": \"https://github.com/ernado\",\n    \"avatar_url\": \"https://avatars.githubusercontent.com/u/866677?\"\n  },\n  \"action\": \"opened\",\n  \"issue\": {\r\n      \"url\": \"https://api.github.com/repos/ernado/oss-estimator/issues/14\",\r\n      \"repository_url\": \"https://api.github.com/repos/ernado/oss-estimator\",\r\n      \"labels_url\": \"https://api.github.com/repos/ernado/oss-estimator/issues/14/labels{/name}\",\r\n      \"comments_url\": \"https://api.github.com/repos/ernado/oss-estimator/issues/14/comments\",\r\n      \"events_url\": \"https://api.github.com/repos/ernado/oss-estimator/issues/14/events\",\r\n      \"html_url\": \"https://github.com/ernado/oss-estimator/issues/14\",\r\n      \"id\": 1637789355,\r\n      \"node_id\": \"I_kwDOJGfUlc5hnq6r\",\r\n      \"number\": 14,\r\n      \"title\": \"test4\",\r\n      \"user\": {\r\n        \"login\": \"ernado\",\r\n        \"id\": 866677,\r\n        \"node_id\": \"MDQ6VXNlcjg2NjY3Nw==\",\r\n        \"avatar_url\": \"https://avatars.githubusercontent.com/u/866677?v=4\",\r\n        \"gravatar_id\": \"\",\r\n        \"url\": \"https://api.github.com/users/ernado\",\r\n        \"html_url\": \"https://github.com/ernado\",\r\n        \"followers_url\": \"https://api.github.com/users/ernado/followers\",\r\n        \"following_url\": \"https://api.github.com/users/ernado/following{/other_user}\",\r\n        \"gists_url\": \"https://api.github.com/users/ernado/gists{/gist_id}\",\r\n        \"starred_url\": \"https://api.github.com/users/ernado/starred{/owner}{/repo}\",\r\n        \"subscriptions_url\": \"https://api.github.com/users/ernado/subscriptions\",\r\n        \"organizations_url\": \"https://api.github.com/users/ernado/orgs\",\r\n        \"repos_url\": \"https://api.github.com/users/ernado/repos\",\r\n        \"events_url\": \"https://api.github.com/users/ernado/events{/privacy}\",\r\n        \"received_events_url\": \"https://api.github.com/users/ernado/received_events\",\r\n        \"type\": \"User\",\r\n        \"site_admin\": false\r\n      },\r\n      \"labels\": [],\r\n      \"state\": \"open\",\r\n      \"locked\": false,\r\n      \"assignee\": null,\r\n      \"assignees\": [],\r\n      \"milestone\": null,\r\n      \"comments\": 0,\r\n      \"created_at\": \"2023-03-23T15:41:09Z\",\r\n      \"updated_at\": \"2023-03-23T15:41:09Z\",\r\n      \"closed_at\": null,\r\n      \"author_association\": \"OWNER\",\r\n      \"active_lock_reason\": null,\r\n      \"body\": null,\r\n      \"reactions\": {\r\n        \"url\": \"https://api.github.com/repos/ernado/oss-estimator/issues/14/reactions\",\r\n        \"total_count\": 0,\r\n        \"+1\": 0,\r\n        \"-1\": 0,\r\n        \"laugh\": 0,\r\n        \"hooray\": 0,\r\n        \"confused\": 0,\r\n        \"heart\": 0,\r\n        \"rocket\": 0,\r\n        \"eyes\": 0\r\n      },\r\n      \"timeline_url\": \"https://api.github.com/repos/ernado/oss-estimator/issues/14/timeline\",\r\n      \"performed_via_github_app\": null,\r\n      \"state_reason\": null\r\n    },\n  \"repository\": {\n    \"id\": 610784405,\n    \"full_name\": \"ernado/oss-estimator\",\n    \"url\": \"https://api.github.com/repos/ernado/oss-estimator\",\n    \"html_url\": \"https://github.com/ernado/oss-estimator\",\n    \"name\": \"oss-estimator\",\n    \"owner\": {\n      \"login\": \"ernado\"\n    }\n  }\n}"

	out := normalizeNewlines([]byte(raw))

	require.Equal(t, normalized, string(out))
}