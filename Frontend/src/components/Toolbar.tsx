interface ToolbarProps {
  onExecute: () => void;
  onClear: () => void;
  onSave: () => void;
  onLoad: () => void;
  isLoading?: boolean;
  connectionStatus?: 'connected' | 'disconnected' | 'checking';
  onToggleFileSystem?: () => void;
  showFileSystemToggle?: boolean;
  fileSystemVisible?: boolean;
}

const Toolbar = ({ onExecute, onClear, onSave, onLoad, isLoading, connectionStatus, onToggleFileSystem, showFileSystemToggle, fileSystemVisible }: ToolbarProps) => {
  const getStatusIcon = () => {
    switch (connectionStatus) {
      case 'connected':
        return '🟢';
      case 'disconnected':
        return '🔴';
      case 'checking':
        return '🟡';
      default:
        return '⚪';
    }
  };

  const getStatusText = () => {
    switch (connectionStatus) {
      case 'connected':
        return 'Conectado al servidor';
      case 'disconnected':
        return 'Desconectado del servidor';
      case 'checking':
        return 'Verificando conexión...';
      default:
        return 'Estado desconocido';
    }
  };

  return (
    <div className="toolbar">
      <div className="toolbar-left">
        <h3>Entrada de Comandos GODISK</h3>
        {connectionStatus && (
          <div className="connection-status" title={getStatusText()}>
            <span className="status-icon">{getStatusIcon()}</span>
            <span className="status-text">{getStatusText()}</span>
          </div>
        )}
      </div>
      
      <div className="toolbar-right">
        <button onClick={onLoad} className="btn btn-secondary" disabled={isLoading}>
          📁 Cargar
        </button>
        <button onClick={onSave} className="btn btn-secondary" disabled={isLoading}>
          💾 Guardar
        </button>
        {showFileSystemToggle && (
          <button 
            onClick={onToggleFileSystem} 
            className={`btn ${fileSystemVisible ? 'btn-primary' : 'btn-secondary'}`}
            disabled={isLoading}
            title={fileSystemVisible ? 'Ocultar explorador de archivos' : 'Mostrar explorador de archivos'}
          >
            {fileSystemVisible ? '🗂️ Ocultar FS' : '🗂️ Mostrar FS'}
          </button>
        )}
        <button onClick={onClear} className="btn btn-danger" disabled={isLoading}>
          🗑️ Limpiar
        </button>
        <button 
          onClick={onExecute} 
          className="btn btn-primary" 
          disabled={isLoading || connectionStatus === 'disconnected'}
        >
          {isLoading ? '⏳ Ejecutando...' : '▶️ Ejecutar'}
        </button>
      </div>
    </div>
  );
};

export default Toolbar;