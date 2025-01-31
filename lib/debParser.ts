import { gunzip } from 'fflate';
import * as fzstd from 'fzstd';
import { LZMA } from 'lzma-purejs';

export interface DebInfo {
  control: {
    package: string;
    version: string;
    architecture: string;
    maintainer: string;
    description: string;
    [key: string]: string | undefined;
  };
  files: string[];
}

interface ArHeader {
  name: string;
  timestamp: number;
  ownerId: number;
  groupId: number;
  mode: number;
  size: number;
}

function parseArHeader(header: Uint8Array): ArHeader {
  const decoder = new TextDecoder();
  
  const name = decoder.decode(header.slice(0, 16)).trim();
  const timestamp = parseInt(decoder.decode(header.slice(16, 28)).trim(), 10);
  const ownerId = parseInt(decoder.decode(header.slice(28, 34)).trim(), 10);
  const groupId = parseInt(decoder.decode(header.slice(34, 40)).trim(), 10);
  const mode = parseInt(decoder.decode(header.slice(40, 48)).trim(), 8);
  const size = parseInt(decoder.decode(header.slice(48, 58)).trim(), 10);
  
  return { name, timestamp, ownerId, groupId, mode, size };
}

async function decompress(compressed: Uint8Array, format: 'gzip' | 'zstd' | 'xz', onProgress?: (status: string) => void): Promise<Uint8Array> {
  onProgress?.(`Starting ${format} decompression...`);

  if (format === 'zstd') {
    try {
      onProgress?.('Decompressing zstd data...');
      const decompressed = fzstd.decompress(compressed);
      onProgress?.('Decompression complete');
      return decompressed;
    } catch (error) {
      console.error('Zstd decompression error:', error);
      throw new Error(`Failed to decompress zstd data: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  if (format === 'xz') {
    try {
      onProgress?.('Decompressing xz data...');
      return new Promise((resolve, reject) => {
        LZMA.decompress(compressed, (result, error) => {
          if (error) {
            reject(new Error(`Failed to decompress xz data: ${error}`));
            return;
          }
          resolve(new Uint8Array(result));
        });
      });
    } catch (error) {
      console.error('XZ decompression error:', error);
      throw new Error(`Failed to decompress xz data: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  return new Promise((resolve, reject) => {
    if (format === 'gzip') {
      gunzip(compressed, (err, decompressed) => {
        if (err) {
          console.error('Decompression error:', err);
          reject(new Error(`Failed to decompress gzip data: ${err.message}`));
          return;
        }
        onProgress?.('Decompression complete');
        resolve(decompressed);
      });
    } else {
      reject(new Error(`Unknown compression format: ${format}`));
    }
  });
}

function parseTarHeader(buffer: Uint8Array, offset: number): { name: string; size: number } | null {
  if (offset + 512 > buffer.length) return null;
  if (buffer.slice(offset, offset + 512).every(byte => byte === 0)) return null;
  
  const decoder = new TextDecoder();
  const name = decoder.decode(buffer.slice(offset, offset + 100)).trim().replace(/\0/g, '');
  const sizeStr = decoder.decode(buffer.slice(offset + 124, offset + 136)).trim();
  
  try {
    const size = parseInt(sizeStr, 8);
    if (isNaN(size)) throw new Error('Invalid size value in tar header');
    return { name, size };
  } catch (error) {
    throw new Error('Failed to parse tar header: Invalid size format');
  }
}

async function extractControlFile(tarData: Uint8Array, onProgress?: (status: string) => void): Promise<string> {
  let offset = 0;
  let filesFound = 0;
  
  onProgress?.('Scanning tar archive for control file...');
  while (offset < tarData.length) {
    const header = parseTarHeader(tarData, offset);
    if (!header) {
      onProgress?.('Reached end of tar archive');
      break;
    }
    
    filesFound++;
    onProgress?.(`Examining file ${filesFound}: ${header.name}`);
    
    offset += 512; // Move past header
    
    if (header.name === './control' || header.name === 'control') {
      onProgress?.('Found control file, extracting...');
      if (offset + header.size > tarData.length) {
        throw new Error('Control file data extends beyond buffer');
      }
      const controlData = tarData.slice(offset, offset + header.size);
      const content = new TextDecoder().decode(controlData);
      onProgress?.('Control file extracted successfully');
      return content;
    }
    
    offset += header.size;
    if (offset % 512 !== 0) {
      offset += 512 - (offset % 512);
    }
  }
  
  throw new Error(`Control file not found in archive (scanned ${filesFound} files)`);
}

export async function parseDebFile(input: File | Buffer | Uint8Array, onProgress?: (status: string) => void): Promise<DebInfo> {
  onProgress?.('Reading file...');
  
  let arrayBuffer: ArrayBuffer;
  if (input instanceof File) {
    arrayBuffer = await input.arrayBuffer();
  } else if (Buffer.isBuffer(input) || input instanceof Uint8Array) {
    arrayBuffer = input.buffer;
  } else {
    throw new Error('Invalid input type: must be File, Buffer, or Uint8Array');
  }

  onProgress?.('Checking file format...');
  const magic = new Uint8Array(arrayBuffer.slice(0, 8));
  const magicStr = new TextDecoder().decode(magic);
  if (!magicStr.startsWith("!<arch>")) {
    throw new Error("Invalid .deb file format: Missing ar archive magic number");
  }

  const debInfo: DebInfo = {
    control: {
      package: '',
      version: '',
      architecture: '',
      maintainer: '',
      description: ''
    },
    files: []
  };

  let offset = 8; // Skip magic number
  let foundControl = false;
  let foundData = false;

  onProgress?.('Parsing package structure...');
  while (offset < arrayBuffer.byteLength) {
    if (offset + 60 > arrayBuffer.byteLength) {
      throw new Error('Unexpected end of file while reading ar header');
    }

    const headerBytes = new Uint8Array(arrayBuffer.slice(offset, offset + 60));
    const header = parseArHeader(headerBytes);
    offset += 60;

    if (offset + header.size > arrayBuffer.byteLength) {
      throw new Error(`Invalid file size in AR archive: ${header.size} bytes`);
    }

    onProgress?.(`Processing archive member: ${header.name}`);
    const content = new Uint8Array(arrayBuffer.slice(offset, offset + header.size));
    
    try {
      if (header.name.startsWith("control.tar")) {
        onProgress?.('Found control.tar, starting extraction...');
        foundControl = true;
        
        let decompressed: Uint8Array;
        if (header.name.endsWith('.gz')) {
          decompressed = await decompress(content, 'gzip', onProgress);
        } else if (header.name.endsWith('.zst')) {
          decompressed = await decompress(content, 'zstd', onProgress);
        } else if (header.name.endsWith('.xz')) {
          decompressed = await decompress(content, 'xz', onProgress);
        } else {
          throw new Error(`Unsupported compression format in ${header.name}`);
        }
        
        const controlContent = await extractControlFile(decompressed, onProgress);
        
        onProgress?.('Parsing control file content...');
        const lines = controlContent.split('\n');
        let currentKey = '';
        
        for (const line of lines) {
          if (!line.trim()) continue;
          
          if (line.startsWith(' ')) {
            if (currentKey && debInfo.control[currentKey]) {
              debInfo.control[currentKey] += '\n' + line.trim();
            }
          } else {
            const colonIndex = line.indexOf(':');
            if (colonIndex > 0) {
              currentKey = line.substring(0, colonIndex).toLowerCase();
              const value = line.substring(colonIndex + 1).trim();
              if (value) {
                debInfo.control[currentKey] = value;
              }
            }
          }
        }
        onProgress?.('Control file parsed successfully');
      } else if (header.name.startsWith("data.tar")) {
        onProgress?.('Found data.tar archive');
        foundData = true;
        debInfo.files.push("(File listing not available - compressed data)");
      }
    } catch (error) {
      console.error(`Error processing ${header.name}:`, error);
      throw new Error(`Failed to process ${header.name}: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
    
    offset += header.size;
    if (offset % 2 !== 0) offset++;
  }

  if (!foundControl) {
    throw new Error('No control.tar.gz found in package');
  }

  if (!foundData) {
    throw new Error('No data.tar.gz found in package');
  }

  if (Object.keys(debInfo.control).length === 0) {
    throw new Error('No control information found in package');
  }

  onProgress?.('Analysis complete');
  return debInfo;
}