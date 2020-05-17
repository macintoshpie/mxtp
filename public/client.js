const host = 'http://mxtp.xyz'
const baseUrl = `${host}/.netlify/functions/jockey`

export async function getLeagueAndThemes(leagueName) {
  return fetch(`${baseUrl}/leagues/${leagueName}`)
    .then(response => response.json())
    .catch(e => ({error: e, message: 'Failed to fetch league info'}))
}

export async function getThemeAndSubmissions(leagueName, themeId) {
  return fetch(`${baseUrl}/leagues/${leagueName}/themes/${themeId}`)
    .then(response => response.json())
    .catch(e => ({error: e, message: 'Failed to fetch theme info'}))
}

export async function updateSubmission(user, leagueName, themeId, songUrl) {
  const data = {
    SongUrl: songUrl
  }
  return fetch(`${baseUrl}/leagues/${leagueName}/themes/${themeId}/submissions`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'authorization': 'Basic ' + btoa(user.username),
    },
    redirect: 'follow',
    body: JSON.stringify(data)
  })
}

export async function updateVotes(user, leagueName, themeId, voteIds) {
  const data = {
    Votes: voteIds
  }
  return fetch(`${baseUrl}/leagues/${leagueName}/themes/${themeId}/votes`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'authorization': 'Basic ' + btoa(user.username),
    },
    redirect: 'follow',
    body: JSON.stringify(data)
  })
}
