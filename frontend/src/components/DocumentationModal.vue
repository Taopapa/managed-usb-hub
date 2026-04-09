<script setup>
import { defineProps, defineEmits, ref } from 'vue'

const props = defineProps({
  show: Boolean
})

const emit = defineEmits(['close'])

const activeTab = ref('gui')
</script>

<template>
  <div v-if="show" class="modal-overlay" @click.self="$emit('close')">
    <div class="modal doc-modal">
      <div class="modal-header">
        <span>User Documentation</span>
        <button class="close-btn" @click="$emit('close')">×</button>
      </div>
      
      <div class="modal-body">
        <div class="doc-tabs">
          <div 
            class="doc-tab" 
            :class="{ active: activeTab === 'gui' }"
            @click="activeTab = 'gui'"
          >
            GUI Guide
          </div>
          <div 
            class="doc-tab" 
            :class="{ active: activeTab === 'cli' }"
            @click="activeTab = 'cli'"
          >
            CLI Guide
          </div>
        </div>

        <div class="doc-content">
          <!-- GUI Documentation -->
          <div v-if="activeTab === 'gui'" class="scrollable-content">
            <h3>Graphical User Interface Guide</h3>
            
            <section>
              <h4>1. Connection & Device Selection</h4>
              <p>The application automatically scans for connected C2G USB Hub Managers. Use the <strong>Device List</strong> in the top section to select a hub to control. If your device doesn't appear, use <strong>File > Scan for USB Hubs</strong>.</p>
            </section>

            <section>
              <h4>2. Port Control</h4>
              <p>The <strong>Control Panel</strong> allows you to toggle individual ports on/off. Click a port button to change its state. The LED indicator shows the current status (Green = On, Grey = Off).</p>
            </section>

            <section>
              <h4>3. Logs & Monitoring</h4>
              <p>Switch to the <strong>Logs</strong> tab to view real-time events for the selected device. You can export these logs to a CSV file via <strong>File > Export Logs</strong>.</p>
            </section>

            <section>
              <h4>4. Security</h4>
              <p>Some operations may require a password if the device is locked. Use <strong>Tools > Clear Stored Password</strong> if you need to reset the session authentication.</p>
            </section>
          </div>

          <!-- CLI Documentation -->
          <div v-if="activeTab === 'cli'" class="scrollable-content">
            <h3>Command Line Interface Guide</h3>
            <p>The application includes a built-in CLI tool (<code>muhcli</code>). Open it via <strong>Tools > Run CLI Command</strong>.</p>
            <p><strong>Usage:</strong> <code>muhcli command [password] [argument]</code></p>

            <section>
              <h4>Query & Info</h4>
              <ul class="cmd-list">
                <li><code>/Q</code> - Query all C2G USB Hubs (no password required).</li>
                <li><code>/Q:COM [-F]</code> - Query a specific COM port or UID. Optional: <code>-F</code> for formatted output.</li>
                <li><code>/G:COM [-B|-H]</code> - Get current port states. Optional: <code>-B</code> (binary), <code>-H</code> (hex).</li>
              </ul>
            </section>

            <section>
              <h4>Port Control (Password Required)</h4>
              <ul class="cmd-list">
                <li><code>/S:COM [pass] [states]</code> - Set port states.</li>
                <li><code>/F:COM [pass] [states]</code> - Set port states and save as initial states.</li>
              </ul>
              <p style="margin-top: 10px; font-weight: bold;">States Format:</p>
              <ul class="cmd-list">
                <li><code>1:3,4</code> - Turn ports 3 and 4 ON.</li>
                <li><code>0:3</code> - Turn port 3 OFF.</li>
                <li><code>T:1,2</code> - Toggle ports 1 and 2.</li>
                <li><code>1:ALL</code> - Turn ALL ports ON.</li>
                <li><code>0:ALL</code> - Turn ALL ports OFF.</li>
                <li><code>B:0101</code> - Binary mask (1=ON, 0=OFF).</li>
                <li><code>H:A601</code> - Hex string (Little-endian).</li>
              </ul>
            </section>

            <section>
              <h4>Configuration & Maintenance</h4>
              <ul class="cmd-list">
                <li><code>/P:COM [old] new</code> - Change password (max 8 chars).</li>
                <li><code>/T:COM [pass]</code> - Get Device Name.</li>
                <li><code>/X:COM [pass] 'name'</code> - Set Device Name (max 8 chars).</li>
                <li><code>/W:COM [pass]</code> - Save current states to flash as initial.</li>
                <li><code>/D:COM [pass]</code> - Restore factory defaults.</li>
                <li><code>/R:COM [pass]</code> - Reset the Hub.</li>
              </ul>
            </section>

            <section>
              <h4>Examples</h4>
              <div class="code-block">
                # Query all hubs<br>
                > muhcli /Q<br><br>
                # Turn on ports 1 and 2 on COM3<br>
                > muhcli /S:COM3 1:1,2<br><br>
                # Turn off port 4 on COM3 (using password '1234')<br>
                > muhcli /S:COM3 1234 0:4<br><br>
                # Query status of COM3 in hex<br>
                > muhcli /G:COM3 -H
              </div>
            </section>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9998;
}

.modal.doc-modal {
  background-color: #fff;
  border: 1px solid #999;
  box-shadow: 2px 2px 10px rgba(0, 0, 0, 0.2);
  display: flex;
  flex-direction: column;
  width: 700px;
  height: 600px;
  max-width: 90vw;
  max-height: 90vh;
}

.modal-header {
  background-color: #0078d7;
  color: #fff;
  padding: 8px 15px;
  font-weight: bold;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.close-btn {
  background: none;
  border: none;
  color: #fff;
  font-size: 20px;
  cursor: pointer;
  padding: 0;
  line-height: 1;
}

.modal-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background-color: #f9f9f9;
}

.doc-tabs {
  display: flex;
  background-color: #e0e0e0;
  border-bottom: 1px solid #ccc;
}

.doc-tab {
  padding: 10px 20px;
  cursor: pointer;
  border-right: 1px solid #ccc;
  background-color: #e0e0e0;
  font-weight: 500;
}

.doc-tab:hover {
  background-color: #eaeaea;
}

.doc-tab.active {
  background-color: #fff;
  border-bottom: 1px solid #fff;
  margin-bottom: -1px;
  color: #0078d7;
}

.doc-content {
  flex: 1;
  overflow: hidden;
  background-color: #fff;
  padding: 0;
  display: flex;
  flex-direction: column;
}

.scrollable-content {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  line-height: 1.6;
}

h3 {
  margin-top: 0;
  color: #0078d7;
  border-bottom: 1px solid #eee;
  padding-bottom: 10px;
}

h4 {
  margin-top: 20px;
  margin-bottom: 10px;
  color: #333;
}

section {
  margin-bottom: 25px;
}

p {
  margin: 0 0 10px 0;
  color: #555;
}

.cmd-list {
  list-style-type: none;
  padding: 0;
}

.cmd-list li {
  margin-bottom: 8px;
  font-family: 'Consolas', monospace;
  background-color: #f5f5f5;
  padding: 5px 10px;
  border-radius: 4px;
  border-left: 3px solid #0078d7;
}

.code-block {
  background-color: #2d2d2d;
  color: #f8f8f2;
  padding: 15px;
  border-radius: 4px;
  font-family: 'Consolas', monospace;
  font-size: 12px;
}
</style>
