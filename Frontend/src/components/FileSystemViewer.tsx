import { useState, useEffect } from 'react';
import React from 'react';
import FileManagerActions from './FileManagerActions';
import FileContentViewer from './FileContentViewer';
import { API_ENDPOINTS } from '../config/api';

interface FileSystemNode {
  name: string;
  type: 'file' | 'folder';
  size?: number;
  path: string;
  children?: FileSystemNode[];
  permissions?: string;
  owner?: string;
  group?: string;
  modified?: string;
}

interface FileSystemViewerProps {
  partitionId: string;
  isVisible: boolean;
}

const FileSystemViewer = ({ partitionId, isVisible }: FileSystemViewerProps) => {
  const [fileSystem, setFileSystem] = useState<FileSystemNode | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [expandedFolders, setExpandedFolders] = useState<Set<string>>(new Set(['/']));
  const [selectedNode, setSelectedNode] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<'tree' | 'list'>('tree');
  const [currentPath, setCurrentPath] = useState('/');
  const [viewingFile, setViewingFile] = useState<{ path: string; name: string } | null>(null);

  const fetchFileSystem = async () => {
    if (!partitionId) return;
    
    setLoading(true);
    setError(null);
    
    try {
      // Usar el nuevo endpoint para obtener el árbol de directorios
      const response = await fetch(API_ENDPOINTS.directoryTree, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        throw new Error(`Error del servidor: ${response.status}`);
      }

      const data = await response.json();
      
      if (!data.success) {
        throw new Error(data.message || 'Error al obtener el árbol de directorios');
      }

            // Función para convertir DirectoryTree del backend a FileSystemNode del frontend
      const convertBackendTree = (backendNode: any, path: string = '/'): FileSystemNode => {
        const nodePath = path === '/' ? `/${backendNode.name}` : `${path}/${backendNode.name}`;
        const actualPath = backendNode.name === 'root' ? '/' : nodePath;
        
        return {
          name: backendNode.name,
          type: backendNode.isDir ? 'folder' : 'file',
          path: actualPath,
          children: backendNode.children ? 
            backendNode.children.map((child: any) => convertBackendTree(child, actualPath)) : 
            undefined,
          permissions: 'rwxr-xr-x', // Valores por defecto por ahora
          owner: 'user',
          group: 'users',
          modified: new Date().toISOString()
        };
      };

      // Procesar la respuesta del backend y convertirla al formato esperado
      const fileSystemTree: FileSystemNode = convertBackendTree(data.tree);
      
      setFileSystem(fileSystemTree);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Error desconocido');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (isVisible && partitionId) {
      fetchFileSystem();
    }
  }, [isVisible, partitionId]);

  const toggleFolder = (path: string) => {
    const newExpanded = new Set(expandedFolders);
    if (newExpanded.has(path)) {
      newExpanded.delete(path);
    } else {
      newExpanded.add(path);
    }
    setExpandedFolders(newExpanded);
  };

  const selectNode = (path: string) => {
    setSelectedNode(path);
  };

  const renderTreeNode = (node: FileSystemNode, level: number = 0) => {
    const isExpanded = expandedFolders.has(node.path);
    const isSelected = selectedNode === node.path;
    const indent = level * 20;

    return (
      <div key={node.path}>
        <div
          className={`fs-node ${isSelected ? 'selected' : ''}`}
          style={{ marginLeft: `${indent}px` }}
          onClick={() => handleNodeClick(node)}
        >
          <div className="fs-node-content">
            {node.type === 'folder' && (
              <button
                className="fs-expand-btn"
                onClick={(e) => {
                  e.stopPropagation();
                  toggleFolder(node.path);
                }}
              >
                {isExpanded ? '📂' : '📁'}
              </button>
            )}
            {node.type === 'file' && (
              <span className="fs-file-icon">📄</span>
            )}
            <span className="fs-node-name">{node.name}</span>
            {node.type === 'file' && node.size && (
              <span className="fs-node-size">({formatFileSize(node.size)})</span>
            )}
          </div>
        </div>
        {node.type === 'folder' && isExpanded && node.children && (
          <div className="fs-children">
            {node.children.map(child => renderTreeNode(child, level + 1))}
          </div>
        )}
      </div>
    );
  };

  const renderListView = (nodes: FileSystemNode[], currentPath: string = '') => {
    const items: React.ReactElement[] = [];
    
    nodes.forEach(node => {
      const fullPath = currentPath ? `${currentPath}/${node.name}` : node.name;
      items.push(
        <tr
          key={node.path}
          className={`fs-list-row ${selectedNode === node.path ? 'selected' : ''}`}
          onClick={() => handleNodeClick(node)}
        >
          <td className="fs-list-name">
            {node.type === 'folder' ? '📁' : '📄'} {node.name}
          </td>
          <td className="fs-list-type">{node.type === 'folder' ? 'Carpeta' : 'Archivo'}</td>
          <td className="fs-list-size">
            {node.type === 'file' && node.size ? formatFileSize(node.size) : '-'}
          </td>
          <td className="fs-list-perms">{node.permissions || '-'}</td>
          <td className="fs-list-owner">{node.owner || '-'}</td>
          <td className="fs-list-modified">{node.modified || '-'}</td>
        </tr>
      );
      
      if (node.children && expandedFolders.has(node.path)) {
        items.push(...renderListView(node.children, fullPath));
      }
    });
    
    return items;
  };

  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const getSelectedNodeDetails = (): FileSystemNode | null => {
    if (!selectedNode || !fileSystem) return null;
    
    const findNode = (node: FileSystemNode): FileSystemNode | null => {
      if (node.path === selectedNode) return node;
      if (node.children) {
        for (const child of node.children) {
          const found = findNode(child);
          if (found) return found;
        }
      }
      return null;
    };
    
    return findNode(fileSystem);
  };

  const handleNodeClick = (node: FileSystemNode) => {
    setSelectedNode(node.path);
    
    if (node.type === 'file') {
      // Doble click para abrir archivo
      setViewingFile({ path: node.path, name: node.name });
    } else if (node.type === 'folder') {
      // Toggle expand/collapse para carpetas
      toggleFolder(node.path);
      setCurrentPath(node.path);
    }
  };

  const refreshFileSystem = () => {
    fetchFileSystem();
  };

  if (!isVisible) return null;

  return (
    <div className="filesystem-viewer">
      <div className="fs-header">
        <div className="fs-header-left">
          <h3>Sistema de Archivos - {partitionId}</h3>
          <div className="fs-view-controls">
            <button
              className={`fs-view-btn ${viewMode === 'tree' ? 'active' : ''}`}
              onClick={() => setViewMode('tree')}
            >
              🌳 Árbol
            </button>
            <button
              className={`fs-view-btn ${viewMode === 'list' ? 'active' : ''}`}
              onClick={() => setViewMode('list')}
            >
              📋 Lista
            </button>
          </div>
        </div>
        <div className="fs-header-right">
          <button
            className="fs-refresh-btn"
            onClick={fetchFileSystem}
            disabled={loading}
          >
            {loading ? '⏳' : '🔄'} Actualizar
          </button>
        </div>
      </div>

      {error && (
        <div className="fs-error">
          ❌ Error: {error}
        </div>
      )}

      <div className="fs-content">
        {loading ? (
          <div className="fs-loading">
            <div className="loading-spinner"></div>
            <span>Cargando sistema de archivos...</span>
          </div>
        ) : fileSystem ? (
          <div className="fs-main">
            <div className="fs-explorer">
              {viewMode === 'tree' ? (
                <div className="fs-tree">
                  {renderTreeNode(fileSystem)}
                </div>
              ) : (
                <div className="fs-list">
                  <table className="fs-list-table">
                    <thead>
                      <tr>
                        <th>Nombre</th>
                        <th>Tipo</th>
                        <th>Tamaño</th>
                        <th>Permisos</th>
                        <th>Propietario</th>
                        <th>Modificado</th>
                      </tr>
                    </thead>
                    <tbody>
                      {fileSystem.children && renderListView(fileSystem.children)}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
            
            {selectedNode && (
              <div className="fs-details">
                <h4>Detalles del Elemento</h4>
                {(() => {
                  const details = getSelectedNodeDetails();
                  if (!details) return <p>No se encontró información</p>;
                  
                  return (
                    <div className="fs-details-content">
                      <div className="fs-detail-item">
                        <strong>Nombre:</strong> {details.name}
                      </div>
                      <div className="fs-detail-item">
                        <strong>Ruta:</strong> {details.path}
                      </div>
                      <div className="fs-detail-item">
                        <strong>Tipo:</strong> {details.type === 'folder' ? 'Carpeta' : 'Archivo'}
                      </div>
                      {details.size && (
                        <div className="fs-detail-item">
                          <strong>Tamaño:</strong> {formatFileSize(details.size)}
                        </div>
                      )}
                      {details.permissions && (
                        <div className="fs-detail-item">
                          <strong>Permisos:</strong> {details.permissions}
                        </div>
                      )}
                      {details.owner && (
                        <div className="fs-detail-item">
                          <strong>Propietario:</strong> {details.owner}
                        </div>
                      )}
                      {details.group && (
                        <div className="fs-detail-item">
                          <strong>Grupo:</strong> {details.group}
                        </div>
                      )}
                      {details.modified && (
                        <div className="fs-detail-item">
                          <strong>Modificado:</strong> {details.modified}
                        </div>
                      )}
                    </div>
                  );
                })()}
              </div>
            )}
          </div>
        ) : (
          <div className="fs-empty">
            <p>📁 Selecciona una partición para ver el sistema de archivos</p>
          </div>
        )}

        {/* File Manager Actions */}
        {partitionId && (
          <FileManagerActions
            partitionId={partitionId}
            currentPath={currentPath}
            onRefresh={refreshFileSystem}
          />
        )}

        {/* File Content Viewer Modal */}
        {viewingFile && (
          <FileContentViewer
            partitionId={partitionId}
            filePath={viewingFile.path}
            fileName={viewingFile.name}
            onClose={() => setViewingFile(null)}
            onRefresh={refreshFileSystem}
          />
        )}
      </div>
    </div>
  );
};

export default FileSystemViewer;