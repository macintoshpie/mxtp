import {
  getLeagueAndThemes,
  getThemeAndSubmissions,
  updateVotes,
  updateSubmission,
} from './client.js'

const state = {
  user: {},
  League: {},
  Themes: [],
  SubmitTheme: {},
  VoteTheme: {},
  VoteThemeSubmissions: [],
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

  addWarning(WARN_FETCHING)
  getLeagueAndThemes("devetry")
    .then(res => {
      if ('error' in res) {
        addWarning(WARN_FETCH_FAILED)
        removeWarning(WARN_FETCHING)
        throw res
      }
      state.League = res.League
      state.Themes = res.Themes
      state.SubmitTheme = res.SubmitTheme
      state.VoteTheme = res.VoteTheme
      return getThemeAndSubmissions("devetry", state.VoteTheme.Id)
    })
    .then(res => {
      if ('error' in res) {
        addWarning(WARN_FETCH_FAILED)
        removeWarning(WARN_FETCHING)
        throw res
      }
      state.VoteThemeSubmissions = res.Submissions
      removeWarning(WARN_FETCHING)

      renderCards()
    })
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
  document.getElementById('submit-theme-form-submit').addEventListener('click', sendSubmission)

  // render vote theme card
  document.getElementById('vote-theme-name').innerText = state.VoteTheme.Name
  document.getElementById('vote-theme-desc').innerText = state.VoteTheme.Description
  const voteThemeForm = document.getElementById('vote-theme-form')
  if (state.VoteThemeSubmissions.length > 0) {
    // create a checkbox for each submission
    state.VoteThemeSubmissions.forEach(sub => {
      const subLabel = document.createElement('label')
      subLabel.for = sub.UserId
      subLabel.innerText = sub.SongUrl
      voteThemeForm.appendChild(subLabel)

      const subInput = document.createElement('input')
      subInput.type = 'checkbox'
      subInput.id = sub.UserId
      subInput.value = sub.UserId

      const subContainer = document.createElement('span')
      subContainer.classList.add('columnar-form-item')
      subContainer.append(subInput, subLabel)
      voteThemeForm.prepend(subContainer)
    })
    document.getElementById('vote-theme-form-submit').addEventListener('click', sendVotes)
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
    state.user.username = username
    removeWarning(WARN_USERNAME)
  } else {
    state.user.username = null
    addWarning(WARN_USERNAME)
  }

  hideOrDisplay()
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
  buttonLoading(btn, updateSubmission(state.user, state.League.Name, state.SubmitTheme.Id, songUrl))
}

const sendVotes = () => {
  const checkedVotes = Array.from(document.querySelectorAll('input[type="checkbox"]:checked'))
  const voteIds = checkedVotes.map(x => x.id)
  const btn = document.getElementById('vote-theme-form-submit')
  buttonLoading(btn, updateVotes(state.user, state.League.Name, state.VoteTheme.Id, voteIds))
}
