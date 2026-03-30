<script setup>
import { ref, onMounted } from 'vue'
import L from 'leaflet'

const attacks = ref([])
const leaderboards = ref([])
let map = null
let markersGroup = null

onMounted(() => {
  initMap()
  pollThreats()
  pollStats()
  
  // Refresh live feed every 2 seconds
  setInterval(pollThreats, 2000)
  // Refresh stats every 10 seconds
  setInterval(pollStats, 10000)
})

const initMap = () => {
  // Center world roughly
  map = L.map('threat-map', {
    center: [20, 0],
    zoom: 2,
    zoomControl: false,
    attributionControl: false
  })

  // Dark-mode map tiles specifically designed for Threat Intelligence Dashboards
  L.tileLayer('https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png', {
    subdomains: 'abcd',
    maxZoom: 19
  }).addTo(map)

  markersGroup = L.layerGroup().addTo(map)
}

const pollThreats = async () => {
  try {
    const res = await fetch('/api/threats/live?limit=30')
    const data = await res.json()
    
    if (data && data.length > 0 && attacks.value.length > 0 && data[0].id === attacks.value[0].id) {
       return // No new attacks to redraw
    }

    attacks.value = data || []
    drawMarkers()
  } catch(e) {
    console.error("Failed to fetch threat feed:", e)
  }
}

const pollStats = async () => {
  try {
    const res = await fetch('/api/threats/stats')
    const data = await res.json()
    leaderboards.value = data.top_countries || []
  } catch(e) {
    console.error("Failed to fetch stats:", e)
  }
}

const drawMarkers = () => {
  if (!map || !markersGroup) return

  markersGroup.clearLayers()
  
  // Create a custom red glowing icon
  const redIcon = L.divIcon({
    className: 'pulsating-circle',
    iconSize: [12, 12]
  })

  attacks.value.forEach(attack => {
    // Only plot if coordinates aren't empty (e.g. from local tests)
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
  <header>
    <div class="pulsating-circle"></div>
    <h1>OVERWATCH <span style="font-weight: 300; opacity: 0.5;">| Cyber Threat Intelligence Engine</span></h1>
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
</template>
