class Creator {
  final int id;
  final String name;
  final String birth;
  final String death;
  final String desc;
  final String image;
  final List<String> alias;
  final int cvId;
  final int gcdId;
  final String resourceUrl;
  final String modified;

  Creator({
    required this.id,
    required this.name,
    required this.birth,
    required this.death,
    required this.desc,
    required this.image,
    required this.alias,
    required this.cvId,
    required this.gcdId,
    required this.resourceUrl,
    required this.modified,
  });

  factory Creator.fromJson(Map<String, dynamic> json) {
    return Creator(
      id: json['id'],
      name: json['name'],
      birth: json['birth'],
      death: json['death'],
      desc: json['desc'],
      image: json['image'],
      alias: (json['teams'] as List<String>),
      cvId: json['cv_id'],
      gcdId: json['gcd_id'],
      resourceUrl: json['resource_url'],
      modified: json['modified'],
    );
  }
}
