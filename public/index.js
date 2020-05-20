import {
  updateVotes,
  updateSubmission,
  getCurrentGame,
} from './client.js'

const state = {
  user: {},
  League: {},
  Themes: [],
  SubmitTheme: {},
  VoteTheme: {},
  VoteThemeSongs: [],
  VoteThemeVotes: [],
  warnings: {
    'warn-card-username': false,
    'warn-card-fetching': false,
    'warn-card-fetchfailed': false,
  },
}

const
  WARN_FETCHING = 'fetching',
  WARN_USERNAME = 'username',
  WARN_FETCH_FAILED = 'fetchfailed'

window.onload = async () => {
  // setup the page
  document.getElementById('who-submit').addEventListener('click', setUser)
  document.getElementById('submit-theme-form-submit').addEventListener('click', sendSubmission)
  document.getElementById('vote-theme-form-submit').addEventListener('click', sendVotes)
}

const loadGameData = async () => {
  addWarning(WARN_FETCHING)

  const response = await getCurrentGame("devetry", state.user.username);
  debugger;
  if ('error' in response) {
    addWarning(WARN_FETCH_FAILED)
    removeWarning(WARN_FETCHING)
    throw response
  }

  state.League = response.League
  state.SubmitTheme = state.League.SubmitTheme
  state.SubmitThemeSong = response.SubmitThemeItems.Songs[0]
  state.VoteTheme = state.League.VoteTheme
  state.VoteThemeSongs = response.VoteThemeItems.Songs
  state.VoteThemeVotes = response.VoteThemeItems.Votes[0]
  
  removeWarning(WARN_FETCHING)
  renderCards()
}

const addWarning = (warning) => {
  state.warnings[`warn-card-${warning}`] = true
  hideOrDisplay()
}

const removeWarning = (warning) => {
  state.warnings[`warn-card-${warning}`] = false
  hideOrDisplay()
}

const renderCards = () => {
  // render submit theme card
  document.getElementById('submit-theme-name').innerText = state.SubmitTheme.Name
  document.getElementById('submit-theme-desc').innerText = state.SubmitTheme.Description
  if (state.SubmitThemeSong != undefined) {
    document.getElementById('submit-theme-form-input').value = state.SubmitThemeSong.SongUrl
  } else {
    document.getElementById('submit-theme-form-input').value = ""
  }

  // render vote theme card
  document.getElementById('vote-theme-name').innerText = state.VoteTheme.Name
  document.getElementById('vote-theme-desc').innerText = state.VoteTheme.Description
  const voteThemeForm = document.getElementById('vote-theme-form')
  // remove any preexisting form checkboxes (can happen if reset user name)
  Array.from(voteThemeForm.getElementsByClassName('vote-theme-form-item')).forEach(e => e.remove())
  if (state.VoteThemeSongs.length > 0) {
    // create a checkbox for each submission
    state.VoteThemeSongs.forEach(sub => {
      const subLabel = document.createElement('label')
      subLabel.for = sub.SubmissionId
      subLabel.innerText = sub.SongUrl
      voteThemeForm.appendChild(subLabel)

      const subInput = document.createElement('input')
      subInput.type = 'checkbox'
      subInput.id = sub.SubmissionId
      subInput.value = sub.SubmissionId
      subInput.checked = state.VoteThemeVotes.SubmissionIds && state.VoteThemeVotes.SubmissionIds.includes(sub.SubmissionId)

      const subContainer = document.createElement('span')
      subContainer.classList.add('columnar-form-item')
      subContainer.classList.add('vote-theme-form-item')
      subContainer.append(subInput, subLabel)
      voteThemeForm.prepend(subContainer)
    })
  } else {
    voteThemeForm.innerText = 'No submissions to vote on...'
  }
}

const hideOrDisplay = () => {
  // warnings is true if at least one warning should be shown
  const warnings = Object.values(state.warnings).some(x => x)
  document.getElementById('submit-card').hidden = state.user.username == undefined || warnings
  document.getElementById('vote-card').hidden = state.user.username == undefined || warnings

  // handle warnings
  document.getElementById('warn-card').hidden = !warnings
  Object.keys(state.warnings).forEach(warnId => {
    document.getElementById(warnId).hidden = !state.warnings[warnId]
  })
}

const userRegex = /[a-z]+@devetry.com/
const setUser = () => {
  const whoInput = document.getElementById('who-input')
  const username = whoInput.value.toLowerCase()
  whoInput.value = username
  if (userRegex.test(username)) {
    if (username == state.user.username) {
      return
    }
    state.user.username = username
    removeWarning(WARN_USERNAME)
  } else {
    state.user.username = null
    addWarning(WARN_USERNAME)
  }

  hideOrDisplay()
  if (state.user.username) {
    loadGameData()
  }
}

const buttonLoading = async (btn, res) => {
  btn.disabled = true
  const originalText = btn.innerText
  btn.innerText = '...'
  await res
  if ('error' in res) {
    btn.innerText = 'failed...'
    btn.classList.add('btn-error')
  } else {
    btn.innerText = 'success'
    btn.classList.add('btn-success')
  }
  btn.disabled = false
  setTimeout(() => {
    btn.classList.remove('btn-success')
    btn.innerText = originalText
  }, 2000)
  return res
}

const sendSubmission = () => {
  const songUrl = document.getElementById('submit-theme-form-input').value
  const btn = document.getElementById('submit-theme-form-submit')
  buttonLoading(btn, updateSubmission(state.user, state.League.Name, state.SubmitTheme.Date, songUrl))
}

const sendVotes = () => {
  const checkedVotes = Array.from(document.querySelectorAll('input[type="checkbox"]:checked'))
  const voteIds = checkedVotes.map(x => x.id)
  const btn = document.getElementById('vote-theme-form-submit')
  buttonLoading(btn, updateVotes(state.user, state.League.Name, state.VoteTheme.Date, voteIds))
}
