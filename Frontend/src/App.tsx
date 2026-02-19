import { useState, useRef, useEffect } from 'react';
import CodeEditor from './components/CodeEditor';
import Toolbar from './components/Toolbar';
import LoginModal from './components/LoginModal';
import FileSystemViewer from './components/FileSystemViewer';
import { API_ENDPOINTS } from './config/api';
import './App.css';

interface LoginCredentials {
  partitionId: string;
  username: string;
  password: string;
  remember: boolean;
}

interface UserSession {
  partitionId: string;
  username: string;
}

function App() {
  const [code, setCode] = useState('# Escribe tus instrucciones GODISK aquí\n# Ejemplos de comandos:\n# mkdisk -size=3000 -unit=M -path=/home/disk1.dsk\n# fdisk -size=1000 -path=/home/disk1.dsk -name=Particion1\n# mount -path=/home/disk1.dsk -name=Particion1\n# mkfs -id=141A -type=ext3\n# mkdir -path="/carpeta1" -id=141A\n# mkfile -path="/archivo1.txt" -size=100 -id=141A');
  const [output, setOutput] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [isLoginModalOpen, setIsLoginModalOpen] = useState(false);
  const [userSession, setUserSession] = useState<UserSession | null>(null);
  const [connectionStatus, setConnectionStatus] = useState<'connected' | 'disconnected' | 'checking'>('disconnected');
  const [showFileSystem, setShowFileSystem] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Check server connection status
  const checkServerConnection = async () => {
    setConnectionStatus('checking');
    try {
      const response = await fetch(API_ENDPOINTS.health, {
        method: 'GET',
      });
      
      if (response.ok) {
        const data = await response.json();
        if (data.status === 'ok') {
          setConnectionStatus('connected');
        } else {
          setConnectionStatus('disconnected');
        }
      } else {
        setConnectionStatus('disconnected');
      }
    } catch (error) {
      setConnectionStatus('disconnected');
    }
  };

  // Check connection on component mount and periodically
  useEffect(() => {
    checkServerConnection();
    const interval = setInterval(checkServerConnection, 30000); // Check every 30 seconds
    return () => clearInterval(interval);
  }, []);

  // Check for active session on component mount
  useEffect(() => {
    checkActiveSession();
  }, []);

  const checkActiveSession = async () => {
    try {
      const response = await fetch(API_ENDPOINTS.session, {
        method: 'GET',
      });

      if (response.ok) {
        const data = await response.json();
        
        if (data.logged_in && data.user) {
          const session: UserSession = {
            partitionId: data.user.partition_id,
            username: data.user.name
          };
          
          setUserSession(session);
          setShowFileSystem(true);
          setOutput(`Sesión activa detectada\nPartición: ${session.partitionId}\nUsuario: ${session.username}\n\nVisualizador del sistema de archivos activado.`);
          
          // Guardar en localStorage si no existe
          const savedSession = localStorage.getItem('userSession');
          if (!savedSession) {
            localStorage.setItem('userSession', JSON.stringify(session));
          }
        } else {
          // No hay sesión activa, verificar localStorage para autolog
          const savedSession = localStorage.getItem('userSession');
          if (savedSession) {
            try {
              // Intentar reautenticar si hay credenciales guardadas
              setOutput('No hay sesión activa en el servidor. Use el botón de login para iniciar sesión.');
              localStorage.removeItem('userSession'); // Limpiar sesión obsoleta
            } catch (error) {
              localStorage.removeItem('userSession');
            }
          }
        }
      }
    } catch (error) {
      console.error('Error checking session:', error);
      // Si no se puede conectar al servidor, mantener sesión local si existe
      const savedSession = localStorage.getItem('userSession');
      if (savedSession) {
        try {
          const session = JSON.parse(savedSession);
          setUserSession(session);
          setShowFileSystem(true);
          setOutput(`Sesión local restaurada (servidor no disponible)\nPartición: ${session.partitionId}\nUsuario: ${session.username}\n\nVisualizador del sistema de archivos activado.`);
        } catch (error) {
        }
                      JSON.parse(savedSession);
      }
    }
  };

  const handleLogin = async (credentials: LoginCredentials) => {
    try {
      const response = await fetch(API_ENDPOINTS.login, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          user: credentials.username,
          pass: credentials.password,
          id: credentials.partitionId,
        }),
      });

      const data = await response.json();

      if (!response.ok || data.status !== 'success') {
        throw new Error(data.message || 'Error en el login');
      }

      // Login exitoso
      const session: UserSession = {
        partitionId: credentials.partitionId,
        username: credentials.username
      };
      
      setUserSession(session);
      
      if (credentials.remember) {
        localStorage.setItem('userSession', JSON.stringify(session));
      }
      
      // Mostrar el visualizador del sistema de archivos después del login
      setShowFileSystem(true);
      
      setOutput(`Sesión iniciada exitosamente\nPartición: ${credentials.partitionId}\nUsuario: ${credentials.username}\n\n${data.message}\n\nVisualizador del sistema de archivos activado.`);
    } catch (error) {
      console.error('Login error:', error);
      throw new Error(error instanceof Error ? error.message : 'Error en el login');
    }
  };

  const handleLogout = async () => {
    try {
      const response = await fetch(API_ENDPOINTS.logout, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      const data = await response.json();
      
      if (response.ok && data.status === 'success') {
        setOutput(`${data.message}\n\nVisualizador del sistema de archivos desactivado.`);
      } else {
        setOutput('Sesión cerrada localmente (error en servidor)');
      }
    } catch (error) {
      console.error('Logout error:', error);
      setOutput('Sesión cerrada localmente (servidor no disponible)');
    } finally {
      // Siempre cerrar sesión localmente
      setUserSession(null);
      localStorage.removeItem('userSession');
      setShowFileSystem(false);
    }
  };

  const toggleFileSystemViewer = () => {
    setShowFileSystem(!showFileSystem);
  };

  const handleExecute = async () => {
    if (!code.trim()) {
      setOutput('Error: No hay código para ejecutar');
      return;
    }

    setIsLoading(true);
    try {
      const response = await fetch(API_ENDPOINTS.analizar, {
        method: 'POST',
        headers: {
          'Content-Type': 'text/plain',
        },
        body: code,
      });
      
      if (!response.ok) {
        setConnectionStatus('disconnected');
        throw new Error(`Error del servidor: ${response.status} ${response.statusText}`);
      }
      
      setConnectionStatus('connected');
      const result = await response.json();
      
      // Format the response in a more readable way
      let formattedOutput = '';
      
      formattedOutput += `═══════════════════════════════════════════════════════════════\n`;
      formattedOutput += `📊 RESUMEN DE EJECUCIÓN\n`;
      formattedOutput += `═══════════════════════════════════════════════════════════════\n`;
      formattedOutput += `• Líneas en total: ${result["Lineas en total"] || 0}\n`;
      formattedOutput += `• Líneas procesadas: ${result["Lineas procesadas"] || 0}\n`;
      formattedOutput += `• Errores encontrados: ${result["Errores encontrados"] || 0}\n\n`;

      if (result.Resultados && result.Resultados.length > 0) {
        formattedOutput += `✅ RESULTADOS:\n`;
        formattedOutput += `═══════════════════════════════════════════════════════════════\n`;
        result.Resultados.forEach((resultado: string) => {
          formattedOutput += `${resultado}\n`;
        });
        formattedOutput += '\n';
      }

      if (result.Errores && result.Errores.length > 0) {
        formattedOutput += `❌ ERRORES:\n`;
        formattedOutput += `═══════════════════════════════════════════════════════════════\n`;
        result.Errores.forEach((error: string) => {
          formattedOutput += `${error}\n`;
        });
      }

      if (!result.Resultados && !result.Errores) {
        formattedOutput += `ℹ️  No se generaron resultados ni errores.\n`;
      }

      setOutput(formattedOutput);
    } catch (error) {
      setConnectionStatus('disconnected');
      let errorMessage = '';
      
      if (error instanceof TypeError && error.message.includes('fetch')) {
        errorMessage = `❌ ERROR DE CONEXIÓN\n`;
        errorMessage += `═══════════════════════════════════════════════════════════════\n`;
        errorMessage += `No se pudo conectar con el servidor backend.\n\n`;
        errorMessage += `Verificar:\n`;
        errorMessage += `• El servidor backend está ejecutándose en ${API_ENDPOINTS.health.replace('/health', '')}\n`;
        errorMessage += `• No hay problemas de firewall o proxy\n`;
        errorMessage += `• El comando correcto para iniciar el servidor es: go run Servidor.go\n\n`;
        errorMessage += `Error técnico: ${error.message}`;
      } else {
        errorMessage = `❌ ERROR\n`;
        errorMessage += `═══════════════════════════════════════════════════════════════\n`;
        errorMessage += `${error instanceof Error ? error.message : String(error)}`;
      }
      
      setOutput(errorMessage);
    } finally {
      setIsLoading(false);
    }
  };

  const handleClear = () => {
    setCode('');
    setOutput('');
  };

  const handleSave = () => {
    const blob = new Blob([code], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'instrucciones.txt';
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleLoad = () => {
    fileInputRef.current?.click();
  };

  const handleFileLoad = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      const reader = new FileReader();
      reader.onload = (e) => {
        const content = e.target?.result as string;
        setCode(content);
      };
      reader.readAsText(file);
    }
  };

  return (
    <div className="app">
      <header className="app-header">
        <div className="header-content">
          <h1>GODISK</h1>
          <div className="user-section">
            {userSession ? (
              <div className="user-info">
                <span className="user-details">
                  {userSession.username} ({userSession.partitionId})
                </span>
                <button onClick={handleLogout} className="login-btn logout">
                  Cerrar Sesión
                </button>
              </div>
            ) : (
              <button onClick={() => setIsLoginModalOpen(true)} className="login-btn">
                Iniciar Sesión
              </button>
            )}
          </div>
        </div>
      </header>
      
      <main className={`app-main ${showFileSystem ? 'with-filesystem' : ''}`}>
        <div className="editor-section">
          <Toolbar
            onExecute={handleExecute}
            onClear={handleClear}
            onSave={handleSave}
            onLoad={handleLoad}
            isLoading={isLoading}
            connectionStatus={connectionStatus}
            onToggleFileSystem={userSession ? toggleFileSystemViewer : undefined}
            showFileSystemToggle={!!userSession}
            fileSystemVisible={showFileSystem}
          />
          <CodeEditor
            initialValue={code}
            onValueChange={setCode}
          />
        </div>
        
        {showFileSystem && userSession && (
          <div className="filesystem-section">
            <FileSystemViewer 
              partitionId={userSession.partitionId} 
              isVisible={showFileSystem}
            />
          </div>
        )}
        
        <div className="output-section">
          <div className="output-header">
            <h3>Resultados de Ejecución</h3>
            {isLoading && <span className="loading">Ejecutando...</span>}
          </div>
          <pre className="output">{output || 'Ejecuta las instrucciones para ver los resultados aquí...'}</pre>
        </div>
      </main>
      
      <LoginModal
        isOpen={isLoginModalOpen}
        onClose={() => setIsLoginModalOpen(false)}
        onLogin={handleLogin}
      />
      
      <input
        type="file"
        ref={fileInputRef}
        onChange={handleFileLoad}
        accept=".txt,.godisk,.smia,.dsk"
        style={{ display: 'none' }}
      />
    </div>
  );
}

export default App;