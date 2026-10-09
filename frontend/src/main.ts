import './style.css'
import App from './App.svelte'
import Overlay from './Overlay.svelte'
import {Mode} from '../wailsjs/go/main/App'

// The same frontend bundle serves both processes; the backend tells us
// which window we are.
Mode().then((mode) => {
  const target = document.getElementById('app')!
  if (mode === 'overlay') {
    document.body.classList.add('overlay')
    new Overlay({target})
  } else {
    new App({target})
  }
})
