import { useEffect, useState } from 'react';
import { ListDirectories, SelectDirectory, Watcher, StopWatching } from "../wailsjs/go/main/App";
import './style.css';

export default function App() {
    const [selectedPath, setSelectedPath] = useState('');
    const [isWatching, setIsWatching] = useState(false);
    const [status, setStatus] = useState('idle'); // 'idle' | 'watching' | 'error'
    const [errorMsg, setErrorMsg] = useState('');

    useEffect(() => {
        // Default to Documents on startup
        ListDirectories().then(dirs => {
            if (dirs.includes('Documents')) {
                setSelectedPath('Documents');
            } else if (dirs.length > 0) {
                setSelectedPath(dirs[0]);
            }
        });
    }, []);

    const handleBrowse = async () => {
        const path = await SelectDirectory();
        if (path) {
            setSelectedPath(path);
        }
    };

    const toggleWatching = () => {
        if (isWatching) {
            StopWatching().then(() => {
                setIsWatching(false);
                setStatus('idle');
            });
            return;
        }

        if (!selectedPath) return;
        
        setIsWatching(true);
        setStatus('watching');
        
        Watcher(selectedPath).catch(err => {
            setIsWatching(false);
            setStatus('error');
            setErrorMsg(String(err));
        });
    };

    // Helper to show a shortened version of the path
    const getDisplayPath = (path) => {
        if (!path) return 'Select a folder...';
        return path;
    };

    return (
        <div className="app">
            <div className="app-wordmark">Organiza</div>

            <div className="app-body">
                <div className={`indicator ${status}`}>
                    <span className="indicator-dot" />
                    <span className="indicator-label">
                        {status === 'idle'     && 'System Idle'}
                        {status === 'watching' && 'Monitoring Desktop'}
                        {status === 'error'    && errorMsg}
                    </span>
                </div>

                <div className="field">
                    <label className="field-label">Destination</label>
                    <div className="path-selector">
                        <div className="path-display" title={selectedPath}>
                            {getDisplayPath(selectedPath)}
                        </div>
                        <button 
                            className="btn-secondary" 
                            onClick={handleBrowse}
                            disabled={isWatching}
                        >
                            Browse
                        </button>
                    </div>
                    <p className="field-hint">
                        Files will be sorted into: <strong>{selectedPath}</strong>
                    </p>
                </div>

                <button
                    className={`btn-primary ${isWatching ? 'btn-danger' : ''}`}
                    onClick={toggleWatching}
                    disabled={!selectedPath && !isWatching}
                >
                    {isWatching ? 'Stop Organizing' : 'Start Organizing'}
                </button>
            </div>
        </div>
    );
}
