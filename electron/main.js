const { app, BrowserWindow, Tray, Menu, nativeImage } = require('electron');
const path = require('path');
const { spawn } = require('child_process');
const fs = require('fs');

let mainWindow;
let tray;
let backendProcess;

// Determine if we're in development or production
const isDev = !app.isPackaged;

// Get the path to the backend executable
function getBackendPath() {
    if (isDev) {
        return path.join(__dirname, '..', 'waddle-backend.exe');
    }
    return path.join(process.resourcesPath, 'waddle-backend.exe');
}

// Get the path to the frontend
function getFrontendPath() {
    if (isDev) {
        return 'http://localhost:5173';
    }
    // In production, frontend is in resources folder
    return path.join(process.resourcesPath, 'frontend', 'dist', 'index.html');
}

// Start the Go backend
function startBackend() {
    const backendPath = getBackendPath();

    if (!fs.existsSync(backendPath)) {
        console.log('Backend executable not found at:', backendPath);
        console.log('Running in frontend-only mode (connect to external backend)');
        return null;
    }

    console.log('Starting backend from:', backendPath);

    backendProcess = spawn(backendPath, [], {
        stdio: ['ignore', 'pipe', 'pipe'],
        detached: false
    });

    backendProcess.stdout.on('data', (data) => {
        console.log(`[Backend] ${data}`);
    });

    backendProcess.stderr.on('data', (data) => {
        console.error(`[Backend Error] ${data}`);
    });

    backendProcess.on('close', (code) => {
        console.log(`Backend process exited with code ${code}`);
    });

    return backendProcess;
}

// Stop the backend
function stopBackend() {
    if (backendProcess) {
        console.log('Stopping backend...');
        backendProcess.kill('SIGTERM');
        backendProcess = null;
    }
}

// Create the main window
function createWindow() {
    // Remove the default menu bar
    Menu.setApplicationMenu(null);

    mainWindow = new BrowserWindow({
        width: 1400,
        height: 900,
        minWidth: 800,
        minHeight: 600,
        icon: path.join(__dirname, 'icon.ico'),
        webPreferences: {
            nodeIntegration: false,
            contextIsolation: true,
            webSecurity: false, // Allow loading local files
        },
        frame: true,
        titleBarStyle: 'default',
        autoHideMenuBar: true, // Hide menu bar
        show: false,
    });

    // Add error handling for page load failures
    mainWindow.webContents.on('did-fail-load', (event, errorCode, errorDescription) => {
        console.error('Failed to load:', errorCode, errorDescription);
    });

    // Load the frontend
    if (isDev) {
        mainWindow.loadURL('http://localhost:5173');
        mainWindow.webContents.openDevTools();
    } else {
        const frontendPath = path.join(process.resourcesPath, 'frontend', 'dist', 'index.html');
        console.log('Loading frontend from:', frontendPath);
        console.log('Frontend exists:', fs.existsSync(frontendPath));

        if (!fs.existsSync(frontendPath)) {
            console.error('Frontend not found! Checking resources directory...');
            try {
                const resourceContents = fs.readdirSync(process.resourcesPath);
                console.log('Resources directory contents:', resourceContents);
            } catch (e) {
                console.error('Could not read resources directory:', e);
            }
        }

        mainWindow.loadFile(frontendPath).catch(err => {
            console.error('Error loading frontend file:', err);
        });
    }

    mainWindow.once('ready-to-show', () => {
        mainWindow.show();
    });

    mainWindow.on('close', (event) => {
        if (!app.isQuitting) {
            event.preventDefault();
            mainWindow.hide();
        }
    });

    mainWindow.on('closed', () => {
        mainWindow = null;
    });
}

// Create system tray
function createTray() {
    const iconPath = path.join(__dirname, 'icon.ico');

    // Create a simple icon if the file doesn't exist
    let trayIcon;
    if (fs.existsSync(iconPath)) {
        trayIcon = nativeImage.createFromPath(iconPath);
    } else {
        // Create a simple 16x16 icon
        trayIcon = nativeImage.createEmpty();
    }

    tray = new Tray(trayIcon);

    const contextMenu = Menu.buildFromTemplate([
        {
            label: 'Open Waddle',
            click: () => {
                if (mainWindow) {
                    mainWindow.show();
                    mainWindow.focus();
                }
            }
        },
        {
            label: 'Recording',
            type: 'checkbox',
            checked: true,
            click: (menuItem) => {
                // Toggle recording status via API
                fetch('http://localhost:8080/api/status', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ paused: !menuItem.checked })
                }).catch(err => console.error('Failed to toggle recording:', err));
            }
        },
        { type: 'separator' },
        {
            label: 'Quit',
            click: () => {
                app.isQuitting = true;
                app.quit();
            }
        }
    ]);

    tray.setToolTip('Waddle - Second Brain');
    tray.setContextMenu(contextMenu);

    tray.on('double-click', () => {
        if (mainWindow) {
            mainWindow.show();
            mainWindow.focus();
        }
    });
}

// App ready
app.whenReady().then(() => {
    // Start backend first
    startBackend();

    // Wait a bit for backend to start, then create window
    setTimeout(() => {
        createWindow();
        createTray();
    }, 2000);
});

// Quit when all windows are closed (except on macOS)
app.on('window-all-closed', () => {
    if (process.platform !== 'darwin') {
        app.quit();
    }
});

app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
        createWindow();
    }
});

// Clean up on quit
app.on('before-quit', () => {
    app.isQuitting = true;
    stopBackend();
});

app.on('quit', () => {
    stopBackend();
});
