interface Result {
  id: number;
  image: string;
  created_at: string;
}

interface UserInfo {
  firstName: string;
  lastName: string;
  username: string;
  email: string;
  avatarSrc: string;
  photos: Result[];
}
