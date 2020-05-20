const baseUrl = location.host === 'www.mxtp.xyz' ? 'https://www.mxtp.xyz' : 'http://localhost:8000'
const apiBaseUrl = `${baseUrl}/.netlify/functions/jockey`

export async function getGame(leagueName, gameId, userId) {
  return fetch(`${apiBaseUrl}/leagues/${leagueName}/games/${gameId}`, {
    headers: {
      'Content-Type': 'application/json',
      'authorization': 'Bearer ' + btoa(userId),
    },
  })
    .then(response => response.json())
    .catch(e => ({error: e, message: 'Failed to fetch game'}))
}

export async function getCurrentGame(leagueName, userId) {
  return getGame(leagueName, 'current', userId)
}

export async function getLeague(leagueName) {
  return fetch(`${apiBaseUrl}/leagues/${leagueName}`)
    .then(response => response.json())
    .catch(e => ({error: e, message: 'Failed to fetch league info'}))
}

export async function getThemeItems(leagueName, themeId) {
  return fetch(`${apiBaseUrl}/leagues/${leagueName}/themes/${themeId}/items`)
    .then(response => response.json())
    .catch(e => ({error: e, message: 'Failed to fetch theme info'}))
}

export async function getThemeSongs(leagueName, themeId) {
  return fetch(`${apiBaseUrl}/leagues/${leagueName}/themes/${themeId}/songs`, {
    headers: {
      'Content-Type': 'application/json',
      'authorization': 'Bearer ' + btoa(user.username),
    },
  })
    .then(response => response.json())
    .catch(e => ({error: e, message: 'Failed to fetch theme info'}))
}

export async function updateSubmission(user, leagueName, themeId, songUrl) {
  const data = {
    SongUrl: songUrl
  }
  return fetch(`${apiBaseUrl}/leagues/${leagueName}/themes/${themeId}/songs`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'authorization': 'Bearer ' + btoa(user.username),
    },
    redirect: 'follow',
    body: JSON.stringify(data)
  })
}

export async function updateVotes(user, leagueName, themeId, submissionIds) {
  const data = {
    SubmissionIds: submissionIds
  }
  return fetch(`${apiBaseUrl}/leagues/${leagueName}/themes/${themeId}/votes`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'authorization': 'Bearer ' + btoa(user.username),
    },
    redirect: 'follow',
    body: JSON.stringify(data)
  })
}
