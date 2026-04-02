<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import L from 'leaflet'

const token = ref(localStorage.getItem('OverwatchJWT') || null)
const username = ref('admin')
const password = ref('')
const loginError = ref('')

const attacks = ref([])
const leaderboards = ref([])
let map = null
let markersGroup = null
let statsInterval = null
let ws = null

onMounted(() => {
  if (token.value) {
    startDashboard()
  }
})

onUnmounted(() => {
  clearInterval(statsInterval)
  if (ws) ws.close()
})

const handleLogin = async () => {
  loginError.value = ''
  try {
    const res = await fetch('/api/login', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({username: username.value, password: password.value})
    })
    const data = await res.json()
    if (res.ok && data.token) {
      token.value = data.token
      localStorage.setItem('OverwatchJWT', data.token)
      password.value = ''
      startDashboard()
    } else {
      loginError.value = data.error || "Authentication Failed"
    }
  } catch(e) {
    loginError.value = "Server unreachable."
  }
}

const handleLogout = () => {
  token.value = null
  localStorage.removeItem('OverwatchJWT')
  if (ws) { ws.close(); ws = null }
  clearInterval(statsInterval)
  attacks.value = []
  leaderboards.value = []
  if (map) {
    map.remove()
    map = null
    markersGroup = null
  }
}

// Ensure the map container is fully rendered before injecting Leaflet
const startDashboard = async () => {
  await nextTick() 
  initMap()
  connectWebSocket()
  pollThreats() // Initial load from REST
  pollStats()
  statsInterval = setInterval(pollStats, 10000)
}

const connectWebSocket = () => {
  const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${protocol}//${location.host}/ws?token=${token.value}`
  
  ws = new WebSocket(wsUrl)

  ws.onmessage = (event) => {
    try {
      const attack = JSON.parse(event.data)
      attacks.value.unshift(attack)
      // Keep only the last 50 attacks in memory
      if (attacks.value.length > 50) attacks.value.pop()
      drawMarkers()
    } catch(e) {
      console.error('WS parse error:', e)
    }
  }

  ws.onclose = () => {
    console.log('WebSocket disconnected. Falling back to polling.')
    // Graceful degradation: fall back to polling if socket dies
    if (token.value) {
      setTimeout(connectWebSocket, 3000) // Auto-reconnect after 3s
    }
  }

  ws.onerror = (err) => {
    console.error('WebSocket error:', err)
  }
}

const initMap = () => {
  if (map) return
  map = L.map('threat-map', {
    center: [20, 0],
    zoom: 2,
    zoomControl: false,
    attributionControl: false
  })

  L.tileLayer('https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png', {
    subdomains: 'abcd',
    maxZoom: 19
  }).addTo(map)

  markersGroup = L.layerGroup().addTo(map)
}

// Utility to inject Authorization Header on all restricted endpoints
const fetchWithAuth = async (url) => {
  const res = await fetch(url, {
    headers: { 'Authorization': `Bearer ${token.value}` }
  })
  if (res.status === 401) {
    handleLogout() // Token expired or invalid, force re-login
    throw new Error("Unauthorized")
  }
  return res.json()
}

const pollThreats = async () => {
  try {
    const data = await fetchWithAuth('/api/threats/live?limit=30')
    if (data && data.length > 0 && attacks.value.length > 0 && data[0].id === attacks.value[0].id) {
       return // No new attacks
    }
    attacks.value = data || []
    drawMarkers()
  } catch(e) {
    console.error("Threat feed error:", e)
  }
}

const pollStats = async () => {
  try {
    const data = await fetchWithAuth('/api/threats/stats')
    leaderboards.value = data.top_countries || []
  } catch(e) {
    console.error("Stats error:", e)
  }
}

const drawMarkers = () => {
  if (!map || !markersGroup) return

  markersGroup.clearLayers()
  const redIcon = L.divIcon({
    className: 'pulsating-circle',
    iconSize: [12, 12]
  })

  attacks.value.forEach(attack => {
    if (attack.lat !== 0 && attack.lon !== 0) {
      L.marker([attack.lat, attack.lon], { icon: redIcon })
        .bindPopup(`<b style="color:black">${attack.ip}</b><br><span style="color:black">${attack.city}, ${attack.country}</span><br><br><span style="color:black">Target: ${attack.target}</span>`)
        .addTo(markersGroup)
    }
  })
}

const formatTime = (ts) => {
  const d = new Date(ts);
  return d.toLocaleTimeString();
}
</script>

<template>
  <!-- Administrator Login Gate -->
  <div v-if="!token" class="login-wrapper">
    <div class="login-box panel">
      <div class="pulsating-circle" style="display:inline-block; margin-bottom: 20px;"></div>
      <h2>OVERWATCH AUTHENTICATION</h2>
      <p style="opacity: 0.6; margin-bottom: 30px;">Threat Intelligence Engine V2</p>
      
      <form @submit.prevent="handleLogin">
        <div class="input-group">
          <label>Administrator ID</label>
          <input type="text" v-model="username" required autocomplete="username" placeholder="admin">
        </div>
        <div class="input-group">
          <label>Encryption Key</label>
          <input type="password" v-model="password" required autocomplete="current-password" placeholder="••••••••">
        </div>
        <div v-if="loginError" class="status terminal-path" style="margin-bottom: 15px; color: #ef4444;">{{ loginError }}</div>
        <button type="submit">Establish Secure Connection</button>
      </form>
    </div>
  </div>

  <!-- Authenticated Secured Dashboard View -->
  <div v-else class="dashboard-wrapper">
    <header>
      <div class="header-left">
        <div class="pulsating-circle"></div>
        <h1>OVERWATCH <span style="font-weight: 300; opacity: 0.5;">| Cyber Threat Intelligence Engine</span></h1>
      </div>
      <button @click="handleLogout" class="logout-btn">Terminate Session</button>
    </header>
    
    <div class="map-container" id="threat-map"></div>

    <div class="sidebar">
      <div class="panel stats-panel">
        <h2>Top Attacking Countries</h2>
        <div v-if="leaderboards.length === 0" class="status" style="text-align:center;">Waiting for payload...</div>
        <div class="leaderboard-row" v-for="stat in leaderboards" :key="stat.country">
          <span>{{ stat.country }}</span>
          <span class="status">{{ stat.count }} attacks</span>
        </div>
      </div>

      <div class="panel terminal-panel" id="terminal">
        <h2>Live Trapped Payloads</h2>
        <div v-if="attacks.length === 0" class="status terminal-wait">
          <div class="pulsating-circle" style="display:inline-block; vertical-align:middle; width:8px; height:8px;"></div> 
          LISTENING FOR BOTNET ACTIVITY ON PORT 80...
        </div>
        <div class="terminal-line" v-for="attack in attacks" :key="attack.id">
          <span class="terminal-ip">[{{ formatTime(attack.timestamp) }}] {{ attack.ip }} ({{ attack.country }})</span><br/>
          <span class="terminal-path">Attempted Exploit: {{ attack.target }}</span><br/>
          <span class="terminal-payload">{{ attack.payload }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Locked Login Box Styling */
.login-wrapper {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background-color: #0b0f19;
}
.login-box {
  width: 100%;
  max-width: 400px;
  text-align: center;
  padding: 40px;
}
.input-group {
  margin-bottom: 20px;
  text-align: left;
}
.input-group label {
  display: block;
  margin-bottom: 8px;
  color: #8da2bb;
  font-size: 0.85rem;
  letter-spacing: 1px;
  text-transform: uppercase;
}
.input-group input {
  width: 100%;
  padding: 12px;
  background: rgba(0, 0, 0, 0.4);
  border: 1px solid #1f2937;
  border-radius: 4px;
  color: #fff;
  font-family: inherit;
  font-size: 1rem;
}
.input-group input:focus {
  outline: none;
  border-color: #3b82f6;
}
button {
  width: 100%;
  padding: 14px;
  background: #1e3a8a;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-family: 'Inter', system-ui, sans-serif;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 1px;
  transition: all 0.2s ease;
}
button:hover {
  background: #2563eb;
}

/* Dashboard specific alignment */
.header-left {
  display: flex;
  align-items: center;
  gap: 15px;
}
.logout-btn {
  width: auto;
  padding: 8px 16px;
  background: transparent;
  border: 1px solid #ef4444;
  color: #ef4444;
  font-size: 0.8rem;
}
.logout-btn:hover {
  background: rgba(239, 68, 68, 0.1);
}
.dashboard-wrapper {
  display: grid;
  height: 100vh;
  grid-template-columns: 2fr 1fr;
  grid-template-rows: auto 1fr;
  grid-template-areas:
    "header header"
    "map sidebar";
  gap: 1rem;
  padding: 1rem;
  box-sizing: border-box;
}
</style>
