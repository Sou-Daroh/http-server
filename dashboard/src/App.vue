<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import L from 'leaflet'

const token = ref(localStorage.getItem('OverwatchJWT') || null)
const username = ref('admin')
const password = ref('')
const loginError = ref('')

const attacks = ref([])
const leaderboards = ref([])
const attackCount = ref(0)
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

const handleSimulate = async () => {
  try {
    await fetch('/api/simulate', {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${token.value}` }
    })
  } catch(e) {
    console.error('Simulation trigger failed:', e)
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
      attackCount.value++
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
      <div class="header-right">
        <div class="attack-badge" :class="{ 'badge-flash': attackCount > 0 }">
          <span class="badge-icon">⚡</span>
          <span class="badge-count">{{ attackCount }}</span>
          <span class="badge-label">LIVE INTERCEPTS</span>
        </div>
        <button @click="handleLogout" class="logout-btn">Terminate Session</button>
      </div>
    </header>
    
    <div class="map-container" id="threat-map"></div>

    <div class="sidebar">
      <button @click="handleSimulate" class="simulate-btn">
        <span class="pulsating-circle" style="background:#ef233c; display:inline-block; margin-right:8px;"></span>
        INITIATE THREAT SIMULATION
      </button>

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
  background: var(--bg-core);
}
.login-box {
  width: 100%;
  max-width: 420px;
  text-align: center;
  padding: 40px;
  background: var(--glass-panel);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border: 1px solid var(--glass-border);
  border-radius: 16px;
  box-shadow: 0 15px 35px rgba(0,0,0,0.5);
}
.input-group {
  margin-bottom: 24px;
  text-align: left;
}
.input-group label {
  display: block;
  margin-bottom: 8px;
  color: var(--text-muted);
  font-size: 0.85rem;
  letter-spacing: 1px;
  text-transform: uppercase;
  font-weight: 600;
}
.input-group input {
  width: 100%;
  padding: 14px;
  background: rgba(0, 0, 0, 0.4);
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 8px;
  color: #fff;
  font-family: inherit;
  font-size: 1rem;
  transition: border-color 0.3s ease;
  box-sizing: border-box;
}
.input-group input:focus {
  outline: none;
  border-color: #38bdf8;
  box-shadow: 0 0 10px rgba(56, 189, 248, 0.2);
}
button {
  width: 100%;
  padding: 16px;
  background: linear-gradient(135deg, #1e3a8a, #2563eb);
  color: white;
  border: 1px solid var(--glass-border);
  border-radius: 8px;
  cursor: pointer;
  font-family: inherit;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 1.5px;
  transition: all 0.3s ease;
}
button:hover {
  background: linear-gradient(135deg, #2563eb, #3b82f6);
  transform: translateY(-2px);
  box-shadow: 0 5px 15px rgba(37, 99, 235, 0.4);
}

/* Dashboard specific alignment */
.header-left {
  display: flex;
  align-items: center;
  gap: 20px;
}
.logout-btn {
  width: auto;
  padding: 8px 20px;
  background: transparent;
  border: 1px solid rgba(239, 68, 68, 0.5);
  color: #ef4444;
  font-size: 0.85rem;
  border-radius: 8px;
}
.logout-btn:hover {
  background: rgba(239, 68, 68, 0.15);
  border-color: #ef4444;
  transform: none;
  box-shadow: none;
}
.dashboard-wrapper {
  display: grid;
  height: 100vh;
  grid-template-columns: 2.2fr 1fr;
  grid-template-rows: auto 1fr;
  grid-template-areas:
    "header header"
    "map sidebar";
  gap: 1.5rem;
  padding: 1.5rem;
  box-sizing: border-box;
}
.header-right {
  display: flex;
  align-items: center;
  gap: 20px;
}
.attack-badge {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 16px;
  border-radius: 8px;
  border: 1px solid rgba(239, 35, 60, 0.3);
  background: rgba(239, 35, 60, 0.1);
  font-family: inherit;
  transition: all 0.3s ease;
}
.badge-flash {
  animation: badgePulse 2s infinite;
}
.badge-icon {
  font-size: 1.2rem;
}
.badge-count {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--primary);
  min-width: 24px;
  text-align: center;
}
.badge-label {
  font-size: 0.7rem;
  color: var(--text-muted);
  letter-spacing: 2px;
  text-transform: uppercase;
  font-weight: 600;
}
.simulate-btn {
  background: rgba(239, 35, 60, 0.15);
  border: 1px solid rgba(239, 35, 60, 0.5);
  color: var(--primary);
  margin-bottom: 0px;
  padding: 14px;
  font-size: 0.85rem;
  animation: badgePulse 2s infinite;
}
.simulate-btn:hover {
  background: rgba(239, 35, 60, 0.3);
  box-shadow: 0 0 20px var(--primary-glow);
  transform: scale(1.02);
}
</style>
