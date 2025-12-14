// Preload script for Electron
// This runs in the renderer process before the web page loads

const { contextBridge, ipcRenderer } = require('electron');

// Expose protected methods to the renderer process
contextBridge.exposeInMainWorld('electronAPI', {
    // App info
    getVersion: () => ipcRenderer.invoke('get-version'),
    
    // Window controls
    minimize: () => ipcRenderer.send('window-minimize'),
    maximize: () => ipcRenderer.send('window-maximize'),
    close: () => ipcRenderer.send('window-close'),
    
    // Recording status
    toggleRecording: (paused) => ipcRenderer.send('toggle-recording', paused),
    onRecordingStatus: (callback) => ipcRenderer.on('recording-status', callback),
    
    // Platform info
    platform: process.platform,
    isElectron: true
});

console.log('Waddle preload script loaded');
