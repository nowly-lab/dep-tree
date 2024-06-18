// Code generated by tygo. DO NOT EDIT.

//////////
// source: graph.go

export interface Node {
  id: number /* int64 */;
  isEntrypoint: boolean;
  fileName: string;
  pathBuf: string[];
  group?: string;
  dirName: string;
  loc: number /* int */;
  size: number /* int */;
}
export interface Link {
  from: number /* int64 */;
  to: number /* int64 */;
  isCyclic: boolean;
}
export interface Graph {
  nodes: Node[];
  links: Link[];
  enableGui: boolean;
}
