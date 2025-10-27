import { useState } from 'react';
import { API_ENDPOINTS } from '../config/api';

interface FileManagerActionsProps {
  partitionId: string;
  currentPath: string;
  onRefresh: () => void;
}

const FileManagerActions = ({ partitionId, currentPath, onRefresh }: FileManagerActionsProps) => {
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [createType, setCreateType] = useState<'file' | 'folder'>('folder');
  const [itemName, setItemName] = useState('');
  const [fileSize, setFileSize] = useState('100');
  const [fileContent, setFileContent] = useState('');
  const [isCreating, setIsCreating] = useState(false);

  const handleCreate = async () => {
    if (!itemName.trim()) {
      alert('Por favor, ingresa un nombre válido');
      return;
    }

    setIsCreating(true);
    
    try {
      let command = '';
      const fullPath = currentPath === '/' ? `/${itemName}` : `${currentPath}/${itemName}`;

      if (createType === 'folder') {
        command = `mkdir -path="${fullPath}" -id=${partitionId}`;
      } else {
        command = `mkfile -path="${fullPath}" -size=${fileSize} -id=${partitionId}`;
        if (fileContent.trim()) {
          command += ` -cont="${fileContent}"`;
        }
      }

      const response = await fetch(API_ENDPOINTS.analizar, {
        method: 'POST',
        headers: {
          'Content-Type': 'text/plain',
        },
        body: command,
      });

      if (response.ok) {
        const result = await response.json();
        console.log('Creation result:', result);
        
        // Reset form
        setItemName('');
        setFileSize('100');
        setFileContent('');
        setShowCreateModal(false);
        
        // Refresh the file system view
        onRefresh();
        
        alert(`${createType === 'folder' ? 'Carpeta' : 'Archivo'} creado exitosamente`);
      } else {
        throw new Error(`Error del servidor: ${response.status}`);
      }
    } catch (error) {
      console.error('Error creating item:', error);
      alert(`Error al crear ${createType === 'folder' ? 'la carpeta' : 'el archivo'}: ${error instanceof Error ? error.message : 'Error desconocido'}`);
    } finally {
      setIsCreating(false);
    }
  };

  const handleDelete = async (itemPath: string, itemType: 'file' | 'folder') => {
    const confirmMessage = `¿Estás seguro de que deseas eliminar ${itemType === 'folder' ? 'la carpeta' : 'el archivo'} "${itemPath}"?`;
    
    if (!confirm(confirmMessage)) {
      return;
    }

    try {
      const command = itemType === 'folder' 
        ? `rmdir -path="${itemPath}" -id=${partitionId}`
        : `remove -path="${itemPath}" -id=${partitionId}`;

      const response = await fetch(API_ENDPOINTS.analizar, {
        method: 'POST',
        headers: {
          'Content-Type': 'text/plain',
        },
        body: command,
      });

      if (response.ok) {
        onRefresh();
        alert(`${itemType === 'folder' ? 'Carpeta' : 'Archivo'} eliminado exitosamente`);
      } else {
        throw new Error(`Error del servidor: ${response.status}`);
      }
    } catch (error) {
      console.error('Error deleting item:', error);
      alert(`Error al eliminar: ${error instanceof Error ? error.message : 'Error desconocido'}`);
    }
  };

  return (
    <div className="file-manager-actions">
      <div className="fm-actions-toolbar">
        <button 
          className="fm-action-btn create-folder"
          onClick={() => {
            setCreateType('folder');
            setShowCreateModal(true);
          }}
          title="Crear nueva carpeta"
        >
          📁➕ Nueva Carpeta
        </button>
        
        <button 
          className="fm-action-btn create-file"
          onClick={() => {
            setCreateType('file');
            setShowCreateModal(true);
          }}
          title="Crear nuevo archivo"
        >
          📄➕ Nuevo Archivo
        </button>
      </div>

      {showCreateModal && (
        <div className="modal-overlay">
          <div className="modal-content fm-create-modal">
            <div className="modal-header">
              <h3>Crear {createType === 'folder' ? 'Carpeta' : 'Archivo'}</h3>
              <button 
                className="close-btn"
                onClick={() => setShowCreateModal(false)}
              >
                ×
              </button>
            </div>
            
            <div className="fm-create-form">
              <div className="form-group">
                <label>Nombre:</label>
                <input
                  type="text"
                  value={itemName}
                  onChange={(e) => setItemName(e.target.value)}
                  placeholder={createType === 'folder' ? 'nombre-carpeta' : 'archivo.txt'}
                  autoFocus
                />
              </div>

              <div className="form-group">
                <label>Ruta completa:</label>
                <input
                  type="text"
                  value={currentPath === '/' ? `/${itemName}` : `${currentPath}/${itemName}`}
                  readOnly
                  className="readonly-input"
                />
              </div>

              {createType === 'file' && (
                <>
                  <div className="form-group">
                    <label>Tamaño (bytes):</label>
                    <input
                      type="number"
                      value={fileSize}
                      onChange={(e) => setFileSize(e.target.value)}
                      min="1"
                      max="10000"
                    />
                  </div>

                  <div className="form-group">
                    <label>Contenido inicial (opcional):</label>
                    <textarea
                      value={fileContent}
                      onChange={(e) => setFileContent(e.target.value)}
                      placeholder="Escribe el contenido inicial del archivo..."
                      rows={4}
                    />
                  </div>
                </>
              )}

              <div className="form-actions">
                <button
                  type="button"
                  onClick={() => setShowCreateModal(false)}
                  className="btn btn-secondary"
                  disabled={isCreating}
                >
                  Cancelar
                </button>
                <button
                  type="button"
                  onClick={handleCreate}
                  className="btn btn-primary"
                  disabled={isCreating || !itemName.trim()}
                >
                  {isCreating ? 'Creando...' : `Crear ${createType === 'folder' ? 'Carpeta' : 'Archivo'}`}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default FileManagerActions;