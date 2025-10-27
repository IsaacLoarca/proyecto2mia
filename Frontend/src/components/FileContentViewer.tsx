import { useState, useEffect } from 'react';
import { API_ENDPOINTS } from '../config/api';

interface FileContentViewerProps {
  partitionId: string;
  filePath: string;
  fileName: string;
  onClose: () => void;
  onRefresh?: () => void;
}

const FileContentViewer = ({ partitionId, filePath, fileName, onClose, onRefresh }: FileContentViewerProps) => {
  const [content, setContent] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [editContent, setEditContent] = useState('');
  const [isSaving, setIsSaving] = useState(false);

  const loadContent = async () => {
    setIsLoading(true);
    try {
      const command = `cat -file="${filePath}" -id=${partitionId}`;
      
      const response = await fetch(API_ENDPOINTS.analizar, {
        method: 'POST',
        headers: {
          'Content-Type': 'text/plain',
        },
        body: command,
      });

      if (response.ok) {
        const result = await response.json();
        const fileContent = result.salida || 'Archivo vacío';
        setContent(fileContent);
        setEditContent(fileContent);
      } else {
        throw new Error(`Error del servidor: ${response.status}`);
      }
    } catch (error) {
      console.error('Error loading file content:', error);
      setContent(`Error al cargar el archivo: ${error instanceof Error ? error.message : 'Error desconocido'}`);
    } finally {
      setIsLoading(false);
    }
  };

  const saveContent = async () => {
    setIsSaving(true);
    try {
      const command = `edit -path="${filePath}" -cont="${editContent}" -id=${partitionId}`;
      
      const response = await fetch(API_ENDPOINTS.analizar, {
        method: 'POST',
        headers: {
          'Content-Type': 'text/plain',
        },
        body: command,
      });

      if (response.ok) {
        setContent(editContent);
        setIsEditing(false);
        if (onRefresh) onRefresh();
        alert('Archivo guardado exitosamente');
      } else {
        throw new Error(`Error del servidor: ${response.status}`);
      }
    } catch (error) {
      console.error('Error saving file content:', error);
      alert(`Error al guardar el archivo: ${error instanceof Error ? error.message : 'Error desconocido'}`);
    } finally {
      setIsSaving(false);
    }
  };

  // Load content when component mounts
  useEffect(() => {
    loadContent();
  }, []);

  return (
    <div className="modal-overlay">
      <div className="modal-content file-content-viewer">
        <div className="modal-header">
          <h3>📄 {fileName}</h3>
          <div className="file-actions">
            {!isEditing ? (
              <>
                <button
                  className="btn btn-sm btn-primary"
                  onClick={() => setIsEditing(true)}
                  disabled={isLoading}
                >
                  ✏️ Editar
                </button>
                <button
                  className="btn btn-sm btn-secondary"
                  onClick={loadContent}
                  disabled={isLoading}
                >
                  🔄 Recargar
                </button>
              </>
            ) : (
              <>
                <button
                  className="btn btn-sm btn-success"
                  onClick={saveContent}
                  disabled={isSaving}
                >
                  {isSaving ? 'Guardando...' : '💾 Guardar'}
                </button>
                <button
                  className="btn btn-sm btn-secondary"
                  onClick={() => {
                    setIsEditing(false);
                    setEditContent(content);
                  }}
                  disabled={isSaving}
                >
                  ❌ Cancelar
                </button>
              </>
            )}
            <button className="close-btn" onClick={onClose}>
              ×
            </button>
          </div>
        </div>

        <div className="file-content-body">
          <div className="file-info">
            <span className="file-path">📍 Ruta: {filePath}</span>
          </div>

          <div className="content-container">
            {isLoading ? (
              <div className="loading-content">
                <div className="loading-spinner">⏳</div>
                <p>Cargando contenido...</p>
              </div>
            ) : isEditing ? (
              <textarea
                className="content-editor"
                value={editContent}
                onChange={(e) => setEditContent(e.target.value)}
                placeholder="Escribe el contenido del archivo..."
                rows={20}
                disabled={isSaving}
              />
            ) : (
              <pre className="content-display">
                {content || 'Archivo vacío'}
              </pre>
            )}
          </div>
        </div>

        <div className="modal-footer">
          <div className="file-stats">
            <span>Caracteres: {(isEditing ? editContent : content).length}</span>
            <span>Líneas: {(isEditing ? editContent : content).split('\n').length}</span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default FileContentViewer;